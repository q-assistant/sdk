package update

const UpdateKindConfig = "config"
const UpdateKindTrigger = "trigger"
const UpdateKindDependency = "dependency"

type UpdateKind string

type Update struct {
	Kind   UpdateKind
	Update interface{}
}

type UpdateFunc func(update *Update)
