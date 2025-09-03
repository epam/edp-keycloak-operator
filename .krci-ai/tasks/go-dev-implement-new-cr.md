
# Task: Implement a new Kubernetes Custom Resource

## Description

This guide provides a comprehensive prompt for LLM to implement a new Kubernetes Custom Resource.

## Prerequisites

**IMPORTANT**: Before starting implementation, you must read and fully understand the following documentation:

1. [Operator Best Practices](./.krci-ai/data/operator-best-practices.md) - Apply ALL the Kubernetes operator-specific patterns, architectural principles, CRD design guidelines, and operational practices defined in this document.

## ⚠️ CRITICAL FIRST STEP

**BEFORE ANY IMPLEMENTATION**: You MUST run the `make operator-sdk create api` command first to scaffold the proper structure. See Step 1.0 below for detailed instructions on how to do this.

**DO NOT** manually create files before running this command!

## Overview

You are tasked with implementing a new Kubernetes Custom Resource for the `your-operator` project. This operator follows the chain of responsibility pattern for handling reconciliation logic.

## Implementation Steps

Follow the [Operator SDK Tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/) as the foundation for implementing your controller.

#### 1 Scaffold API and Controller

**Before implementing the controller, ask the user for the CustomResource details:**

1. **Group**: The API group (typically use `v1` for this project)
2. **Version**: The API version (typically `v1alpha1`)
3. **Kind**: The CustomResource kind name (e.g., `KeycloakClient`, `KeycloakUser`, etc.)

Once you have these details, use the Operator SDK to scaffold the basic API and controller structure:

```bash
make operator-sdk create api --group <group> --version <version> --kind <kind> --resource --controller
```

**Example**: If the user wants to create a `KeycloakClient` CustomResource:

```bash
make operator-sdk create api --group v1 --version v1alpha1 --kind KeycloakClient --resource --controller
```

This command will create:

- API types in `api/v1alpha1/`
- Controller skeleton in `internal/controller/`
- Basic RBAC markers

After scaffolding, you'll need to customize the generated code to follow the project's specific patterns described in the sections below.

#### 2 Implement the API Types

Implement your Custom Resource Definition (CRD) spec and status, based on user requirements, in `api/v1alpha1/`:

**Note**: The following examples use `YourResource` as a placeholder. Replace this with the actual resource name you specified during scaffolding.

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// YourResource is the Schema for the yourresources API
type YourResource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   YourResourceSpec   `json:"spec,omitempty"`
    Status YourResourceStatus `json:"status,omitempty"`
}

// YourResourceSpec defines the desired state of YourResource
type YourResourceSpec struct {
    // Add your spec fields here
}

// YourResourceStatus defines the observed state of YourResource
type YourResourceStatus struct {
    // Add your status fields here
}
```

#### 3 Generate Code and Manifests

Run the following commands to generate the necessary code:

```bash
make generate
make manifests
```

#### 4 Implement the Controller

Implement your controller in `internal/controller/yourresource/` following the existing pattern:

**Note**: Replace `YourResource` and `yourresource` with the actual resource name you specified during scaffolding.

```go
package yourresource

import (
    "context"
    "fmt"
    "time"

    "k8s.io/apimachinery/pkg/api/equality"
    k8sErrors "k8s.io/apimachinery/pkg/api/errors"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"

    yourresourceApi "github.com/your-org/your-operator/api/v1" // Replace with your actual module path
)

const (
    defaultRequeueTime = time.Second * 30
    successRequeueTime = time.Minute * 10
    finalizerName      = "yourresource.operator.finalizer.name"
)

// NewReconcileYourResource creates a new ReconcileYourResource with all necessary dependencies.
func NewReconcileYourResource(
    client client.Client,
) *ReconcileYourResource {
    return &ReconcileYourResource{
        client:            client,
    }
}

type ReconcileYourResource struct {
    client client.Client
}

func (r *ReconcileYourResource) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&yourresourceApi.YourResource{}).
        Complete(r)
}

// +kubebuilder:rbac:groups=yourgroup,namespace=placeholder,resources=yourresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=yourgroup,namespace=placeholder,resources=yourresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=yourgroup,namespace=placeholder,resources=yourresources/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

func (r *ReconcileYourResource) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
    log := ctrl.LoggerFrom(ctx)
    log.Info("Reconciling YourResource")

    yourResource := &yourresourceApi.YourResource{}
    if err := r.client.Get(ctx, request.NamespacedName, yourResource); err != nil {
        if k8sErrors.IsNotFound(err) {
            return reconcile.Result{}, nil
        }
        return reconcile.Result{}, err
    }


    if yourResource.GetDeletionTimestamp() != nil {
        if controllerutil.ContainsFinalizer(yourResource, finalizerName) {
            if err = chain.NewRemoveResource().ServeRequest(ctx, yourResource); err != nil {
                return ctrl.Result{}, err
            }

            controllerutil.RemoveFinalizer(yourResource, finalizerName)

            if err = r.client.Update(ctx, yourResource); err != nil {
                return ctrl.Result{}, err
            }
        }

        return ctrl.Result{}, nil
    }

    if controllerutil.AddFinalizer(yourResource, finalizerName) {
        err = r.client.Update(ctx, yourResource)
        if err != nil {
            return ctrl.Result{}, err
        }

  // Get yourResource again to get the updated object
  if err = r.client.Get(ctx, request.NamespacedName, yourResource); err != nil {
   return reconcile.Result{}, err
  }
    }

    oldStatus := yourResource.Status.DeepCopy()

    if err = chain.MakeChain(r.client).ServeRequest(ctx, yourResource); err != nil {
        log.Error(err, "An error has occurred while handling YourResource")

        yourResource.Status.SetError(err.Error())

        if statusErr := r.updateYourResourceStatus(ctx, yourResource, oldStatus); statusErr != nil {
            return reconcile.Result{}, statusErr
        }

        return reconcile.Result{}, err
    }

    yourResource.Status.SetOK()

    if err = r.updateYourResourceStatus(ctx, yourResource, oldStatus); err != nil {
        return reconcile.Result{}, err
    }

    log.Info("Reconciling YourResource is finished")

    return reconcile.Result{
        RequeueAfter: successRequeueTime,
    }, nil
}

func (r *ReconcileYourResource) updateYourResourceStatus(
 ctx context.Context,
 yourResource *yourresourceApi.YourResource,
 oldStatus yourresourceApi.YourResourceStatus,
) error {
    if equality.Semantic.DeepEqual(&yourResource.Status, oldStatus) {
        return nil
    }

    if err := r.client.Status().Update(ctx, yourResource); err != nil {
        return fmt.Errorf("failed to update YourResource status: %w", err)
    }

    return nil
}
```

#### 5 Implement the Chain of Responsibility

Create a chain package in `internal/controller/yourresource/chain/` with the following structure:

1. `chain.go` - Main chain implementation
2. `factory.go` - Chain factory
3. Individual handler files for each step in the chain

**Note**: Replace `yourresource` and `YourResource` with the actual resource name you specified during scaffolding.

Example `chain.go`:

```go
package chain

import (
    "context"
    "sigs.k8s.io/controller-runtime/pkg/client"

    yourApi "github.com/your-org/your-operator/api/v1"
)

type Chain interface {
    ServeRequest(ctx context.Context, yourResource *yourApi.YourResource) error
}

type chain struct {
    handlers []Handler
}

func (c *chain) ServeRequest(ctx context.Context, yourResource *yourApi.YourResource) error {
    for _, handler := range c.handlers {
        if err := handler.ServeRequest(ctx, yourResource); err != nil {
            return err
        }
    }
    return nil
}

type Handler interface {
    ServeRequest(ctx context.Context, yourResource *yourApi.YourResource) error
}

func MakeChain(k8sClient client.Client) Chain {
    return &chain{
        handlers: []Handler{
            // Add your handlers here
        },
    }
}
```

Example handler implementations should follow the pattern of existing handlers in your chain.

#### 6 Register the Controller

Add your controller to `cmd/main.go`:

```go
import (
    yourresourcecontroller "github.com/your-org/your-operator/controllers/yourresource"
)

// In the main function, add:
if err = yourresourcecontroller.NewReconcileYourResource(mgr.GetClient()).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "YourResource")
    os.Exit(1)
}
```

**Note**: Replace `YourResource` with the actual resource name you specified during scaffolding.
