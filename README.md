## Mutating Webhook Base

This repository provides a base from which a Mutating Webhook can easily be developed.

## Repository Components

## How To Use

### 1. Update go.mod

- Change the module name.

- Add any needed modules so that they can be used in the mutating webhook. 

### 2. Update mutate.go
`mutate.go` offers skaffolding that is used to define the functionality of the mutating webhook. 

- The `var` block offers a place to define global variables. Define any variables that will be passed as arguments here.

- The `init()` function is used to initialize the variables declared in the `var` block. Use the [`flag` package](https://golang.org/pkg/flag/) to define the arguments.

- The `CustomMutator` struct offers a quick way to pass variables or other structs into the `mutate(request v1beta1.AdmissionRequest)` function. This is done by implementing the `Mutator` interface in `main.go` which is used by the webserver.
  > You don't need to use this struct. Feel free implement the `Mutator` interface in your own packages.

- The `setup()` function is called at the start of the program. Use it to ininitialize any structs and, most importantly, initialize and assign the stuct implementing `Mutator` to the `mutator` variable.

- The `Mutate(request v1beta1.AdmissionRequest)` is the wrapper for the logic of your mutating webhook. Complete it with the logic you wish to implement.

## Testing
______________________

## Base Mutating Webhook

Ce répertoire fourni une base duquel un Mutating Webhook peut être facilement développer.
