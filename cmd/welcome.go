package cmd

import (
	"fmt"
	"os"

	"github.com/deformal/kastql/cmd/types"
)

func WelcomeMessage() types.WelcomeMessageResponse {
	response := types.WelcomeMessageResponse{}
	if len(os.Args[1:]) <= 0 {
		fmt.Println("No command was passed, was expecting atlease 1 command")
		os.Exit(1)
	}
	message := "KastQl Router welcome \n"
	response.Command = os.Args[1]
	if len(os.Args[2:]) <= 0 {
		response.Flags = nil
	}
	response.Flags = os.Args[2:]
	fmt.Println(message)
	return response
}
