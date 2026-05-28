package executor

// entityRef points at a map[string]any that holds entity key fields and will
// receive the entity's resolved fields after the _entities call returns.
// Because Go maps are reference types, mutating ref.obj automatically updates
// the data that lives in the parent step's result tree.
type entityRef struct {
	obj map[string]any // the entity placeholder map (already has key fields)
}

// collectEntityRefs navigates the step result tree and returns one ref per
// entity instance that needs to be resolved.
//
// mergePath = ["orders", "user"] means:
//   - start at data
//   - walk through data["orders"] (array)
//   - for each element, look at element["user"]
//   - each such "user" value (a map) becomes one entityRef
func collectEntityRefs(data map[string]any, mergePath []string) []entityRef {
	if len(mergePath) == 0 || data == nil {
		return nil
	}

	entityField := mergePath[len(mergePath)-1]
	containerPath := mergePath[:len(mergePath)-1]

	containers := gatherObjects(data, containerPath)

	var refs []entityRef
	for _, c := range containers {
		v := c[entityField]
		switch t := v.(type) {
		case map[string]any:
			refs = append(refs, entityRef{obj: t})
		case []any:
			for _, elem := range t {
				if m, ok := elem.(map[string]any); ok {
					refs = append(refs, entityRef{obj: m})
				}
			}
		}
	}
	return refs
}

// gatherObjects walks data following path and collects all map[string]any
// values it encounters at that depth. Arrays are expanded element-by-element.
func gatherObjects(data map[string]any, path []string) []map[string]any {
	if len(path) == 0 {
		return []map[string]any{data}
	}
	v, ok := data[path[0]]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case map[string]any:
		return gatherObjects(t, path[1:])
	case []any:
		var result []map[string]any
		for _, elem := range t {
			if m, ok := elem.(map[string]any); ok {
				result = append(result, gatherObjects(m, path[1:])...)
			}
		}
		return result
	}
	return nil
}

// mergeInto shallowly copies src fields into dst.
func mergeInto(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}
