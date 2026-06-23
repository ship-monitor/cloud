package paging

type Paging struct {
	Page int `json:"page" validate:"required,min=0"`
	Size int `json:"size" validate:"required,min=1"`
}
