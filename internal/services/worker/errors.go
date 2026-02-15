package worker

import "errors"

var ErrInvalidTask = errors.New("invalid task")
var ErrTaskNotFound = errors.New("task not found")
var ErrTaskAlreadyExists = errors.New("task already exists")
