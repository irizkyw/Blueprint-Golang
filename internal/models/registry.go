package models

var ModelRegistry = []interface{}{
	new(Role),
	new(Saving),
	new(User),
}