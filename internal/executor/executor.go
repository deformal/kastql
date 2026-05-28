package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"

	"go.uber.org/zap"

	"github.com/deformal/kastql/internal/planner"
)

// Executor runs a QueryPlan and returns the merged GraphQL response.
type Executor struct {
	log *zap.Logger
}

// New creates an Executor.
func New(log *zap.Logger) *Executor {
	return &Executor{log: log}
}

// Execute runs every step in the plan and merges the results.
// headers are forwarded as-is to every upstream call.
func (e *Executor) Execute(ctx context.Context, plan *planner.QueryPlan, headers map[string]string) (*Result, error) {
	if len(plan.Steps) == 0 {
		return &Result{Data: map[string]any{}}, nil
	}

	// Partition into root steps (no deps) and dependent steps.
	var roots, deps []*planner.Step
	for _, s := range plan.Steps {
		if len(s.DependsOn) == 0 {
			roots = append(roots, s)
		} else {
			deps = append(deps, s)
		}
	}

	// Execute root steps in parallel.
	type stepOut struct {
		id   string
		data map[string]any
		errs []GQLError
		err  error
	}
	out := make(chan stepOut, len(roots))

	var wg sync.WaitGroup
	for _, s := range roots {
		wg.Add(1)
		go func(step *planner.Step) {
			defer wg.Done()
			data, errs, err := e.callStep(ctx, step, headers, nil)
			out <- stepOut{id: step.ID, data: data, errs: errs, err: err}
		}(s)
	}
	wg.Wait()
	close(out)

	stepData := map[string]map[string]any{}
	var allErrors []GQLError
	for o := range out {
		if o.err != nil {
			allErrors = append(allErrors, GQLError{Message: o.err.Error()})
			continue
		}
		stepData[o.id] = o.data
		allErrors = append(allErrors, o.errs...)
	}

	// Execute dependent steps sequentially in dependency order.
	// Current plans are at most one level deep (root → entity/join).
	for _, step := range deps {
		parentID := step.DependsOn[0]
		parentData, ok := stepData[parentID]
		if !ok {
			// Parent failed — skip dependent.
			continue
		}

		var errs []GQLError
		switch step.Meta.Kind {
		case planner.StepKindEntity:
			errs = e.executeEntityStep(ctx, step, parentData, headers)
		case planner.StepKindJoin:
			errs = e.executeJoinStep(ctx, step, parentData, headers)
		default:
			data, stepErrs, err := e.callStep(ctx, step, headers, nil)
			if err != nil {
				errs = []GQLError{{Message: err.Error()}}
			} else {
				stepData[step.ID] = data
				errs = stepErrs
			}
		}
		allErrors = append(allErrors, errs...)
	}

	// Merge all root step data into the final response.
	merged := map[string]any{}
	for _, s := range roots {
		if data, ok := stepData[s.ID]; ok {
			mergeInto(merged, data)
		}
	}

	var finalErrors []GQLError
	for _, e := range allErrors {
		if e.Message != "" {
			finalErrors = append(finalErrors, e)
		}
	}

	return &Result{Data: merged, Errors: finalErrors}, nil
}

// executeEntityStep resolves federation entity fields by calling _entities on
// the owning service and merging results back into parentData in-place.
func (e *Executor) executeEntityStep(
	ctx context.Context,
	step *planner.Step,
	parentData map[string]any,
	headers map[string]string,
) []GQLError {
	em := step.Meta.Entity

	refs := collectEntityRefs(parentData, step.MergePath)
	if len(refs) == 0 {
		return nil
	}

	// Build the representations list, preserving order.
	representations := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		rep := map[string]any{"__typename": em.TypeName}
		for _, kf := range em.KeyFields {
			rep[kf] = ref.obj[kf]
		}
		representations = append(representations, rep)
	}

	vars := map[string]any{"representations": representations}
	raw, errs, err := e.callStep(ctx, step, headers, vars)
	if err != nil {
		return []GQLError{{Message: fmt.Sprintf("entity step %s: %s", step.ServiceName, err)}}
	}

	entitiesVal, ok := raw["_entities"]
	if !ok {
		return errs
	}
	entities, ok := entitiesVal.([]any)
	if !ok {
		return errs
	}

	// Merge entity data into each ref in order.
	for i, ref := range refs {
		if i >= len(entities) {
			break
		}
		if entityData, ok := entities[i].(map[string]any); ok {
			mergeInto(ref.obj, entityData)
		}
	}
	return errs
}

// executeJoinStep performs a stitching in-memory join by calling the target
// service once per parent object and inserting the result in-place.
func (e *Executor) executeJoinStep(
	ctx context.Context,
	step *planner.Step,
	parentData map[string]any,
	headers map[string]string,
) []GQLError {
	jm := step.Meta.Join

	joinField := step.MergePath[len(step.MergePath)-1]
	containerPath := step.MergePath[:len(step.MergePath)-1]

	containers := gatherObjects(parentData, containerPath)

	var allErrs []GQLError
	for _, container := range containers {
		keyVal := container[jm.ParentKeyField]
		if keyVal == nil {
			continue
		}

		vars := map[string]any{"arg": keyVal}
		raw, errs, err := e.callStep(ctx, step, headers, vars)
		allErrs = append(allErrs, errs...)
		if err != nil {
			allErrs = append(allErrs, GQLError{
				Message: fmt.Sprintf("join step %s: %s", step.ServiceName, err),
			})
			continue
		}
		if result, ok := raw[jm.TargetField]; ok {
			container[joinField] = result
		}
	}
	return allErrs
}

// callStep makes the upstream HTTP call for one plan step.
// extraVars are merged on top of step.Variables.
func (e *Executor) callStep(
	ctx context.Context,
	step *planner.Step,
	headers map[string]string,
	extraVars map[string]any,
) (map[string]any, []GQLError, error) {
	vars := step.Variables
	if len(extraVars) > 0 {
		merged := make(map[string]any, len(vars)+len(extraVars))
		maps.Copy(merged, vars)
		maps.Copy(merged, extraVars)
		vars = merged
	}

	e.log.Debug("calling upstream",
		zap.String("service", step.ServiceName),
		zap.String("url", step.ServiceURL),
		zap.String("kind", string(step.Meta.Kind)),
	)

	resp, err := callUpstream(ctx, e.log, step.ServiceURL, headers, step.Query, vars)
	if err != nil {
		return nil, nil, err
	}

	var data map[string]any
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return nil, resp.Errors, fmt.Errorf("unmarshal data from %s: %w", step.ServiceName, err)
		}
	}

	return data, resp.Errors, nil
}
