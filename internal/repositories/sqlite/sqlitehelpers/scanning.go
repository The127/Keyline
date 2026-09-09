package sqlitehelpers

type Row interface {
	Scan(...interface{}) error
}
