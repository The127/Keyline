package pghelpers

type Row interface {
	Scan(...interface{}) error
}
