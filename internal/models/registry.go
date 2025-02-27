package models

var ModelRegistry = []interface{}{
	new(Payment),
	new(Role),
	new(Saving),
	new(User),
}