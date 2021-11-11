
## How to use it
go run . 

## What it does
creates the boilerplate for adding a new target

what it creates:
```
cmd/
    newTarget
    (DOES NOT UPDATE) triggermesh-controller/main.go 

config/
    301-newTarget.yaml
     (DOES NOT UPDATE) 500-controller.yaml

pkg/targets/adapter/newtarget/
    adapter.go

pkg/targets/reconciler/newtarget/
    reconciler.go
    adapter.go
    controller.go

pkg/api/targets/v1alpha1/
    newtarget_lifecycle.go
    newtarget_types.go
     (DOES NOT UPDATE) register.go
```

One will need to go into the files noted (DOES NOT UPDATE) above and apply the new changes
in each respective file.


