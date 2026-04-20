package keycloakapi_test

import (
	"context"
	"fmt"
	"log"

	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const exampleRealm = "my-realm"

// ExampleNewKeycloakClient demonstrates client construction with password grant.
func ExampleNewKeycloakClient() {
	ctx := context.Background()

	kc, err := keycloakapi.NewKeycloakClient(ctx, "http://localhost:8080", "admin-cli",
		keycloakapi.WithPasswordGrant("admin", "admin"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Use the client to list realms.
	realms, _, err := kc.Realms.GetRealms(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d realms\n", len(realms))
}

// ExampleNewKeycloakClient_clientCredentials demonstrates client construction with client_credentials grant.
func ExampleNewKeycloakClient_clientCredentials() {
	ctx := context.Background()

	_, err := keycloakapi.NewKeycloakClient(ctx, "http://localhost:8080", "my-service-account",
		keycloakapi.WithClientSecret("my-secret"),
	)
	if err != nil {
		log.Fatal(err)
	}
}

// Example_userCRUD demonstrates a Create → Get → Update → Delete user flow.
func Example_userCRUD() {
	ctx := context.Background()

	kc, err := keycloakapi.NewKeycloakClient(ctx, "http://localhost:8080", "admin-cli",
		keycloakapi.WithPasswordGrant("admin", "admin"),
	)
	if err != nil {
		log.Fatal(err)
	}

	realm := exampleRealm

	// Create a user.
	resp, err := kc.Users.CreateUser(ctx, realm, keycloakapi.UserRepresentation{
		Username: ptr.To("jane"),
		Email:    ptr.To("jane@example.com"),
		Enabled:  ptr.To(true),
	})
	if err != nil {
		log.Fatal(err)
	}

	userID := keycloakapi.GetResourceIDFromResponse(resp)

	// Get the user.
	user, _, err := kc.Users.GetUser(ctx, realm, userID)
	if err != nil {
		log.Fatal(err)
	}

	// Update the user.
	user.FirstName = ptr.To("Jane")

	_, err = kc.Users.UpdateUser(ctx, realm, userID, *user)
	if err != nil {
		log.Fatal(err)
	}

	// Delete the user.
	_, err = kc.Users.DeleteUser(ctx, realm, userID)
	if err != nil {
		log.Fatal(err)
	}
}

// Example_errorHandling demonstrates IsNotFound, IsConflict, and SkipConflict patterns.
func Example_errorHandling() {
	ctx := context.Background()

	kc, err := keycloakapi.NewKeycloakClient(ctx, "http://localhost:8080", "admin-cli",
		keycloakapi.WithPasswordGrant("admin", "admin"),
	)
	if err != nil {
		log.Fatal(err)
	}

	realm := exampleRealm

	// Handle "not found" gracefully.
	_, _, err = kc.Users.GetUser(ctx, realm, "nonexistent-id")
	if keycloakapi.IsNotFound(err) {
		fmt.Println("User does not exist — safe to create")
	}

	// Skip "conflict" errors (idempotent create).
	_, err = kc.Roles.CreateRealmRole(ctx, realm, keycloakapi.RoleRepresentation{
		Name: ptr.To("my-role"),
	})
	if err := keycloakapi.SkipConflict(err); err != nil {
		log.Fatal(err) // only fatal on non-conflict errors
	}
}

// Example_authFlowCopyAndModify demonstrates copying a built-in flow, adding an execution,
// and configuring it.
func Example_authFlowCopyAndModify() {
	ctx := context.Background()

	kc, err := keycloakapi.NewKeycloakClient(ctx, "http://localhost:8080", "admin-cli",
		keycloakapi.WithPasswordGrant("admin", "admin"),
	)
	if err != nil {
		log.Fatal(err)
	}

	realm := exampleRealm

	// Copy the built-in "browser" flow.
	_, err = kc.AuthFlows.CopyAuthFlow(ctx, realm, "browser", "my-custom-browser")
	if err != nil {
		log.Fatal(err)
	}

	// Get the new flow's executions.
	executions, _, err := kc.AuthFlows.GetFlowExecutions(ctx, realm, "my-custom-browser")
	if err != nil {
		log.Fatal(err)
	}

	// Change the first execution to REQUIRED.
	if len(executions) > 0 {
		executions[0].Requirement = ptr.To(keycloakapi.AuthExecutionRequirementRequired)

		_, err = kc.AuthFlows.UpdateFlowExecution(ctx, realm, "my-custom-browser", executions[0])
		if err != nil {
			log.Fatal(err)
		}
	}

	// Set the custom flow as the realm's browser flow.
	_, err = kc.Realms.SetRealmBrowserFlow(ctx, realm, "my-custom-browser")
	if err != nil {
		log.Fatal(err)
	}
}
