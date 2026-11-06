package store

import "errors"

// ErrNotFound is returned by Get when the ID does not exist.
var ErrNotFound = errors.New("invoice not found")

// ErrProductNotFound is returned when a product ID does not exist.
var ErrProductNotFound = errors.New("product not found")

// ErrCustomerNotFound is returned when a customer ID does not exist.
var ErrCustomerNotFound = errors.New("customer not found")

// ErrSupplierNotFound is returned when a supplier ID does not exist.
var ErrSupplierNotFound = errors.New("supplier not found")
