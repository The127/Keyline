package change

type Type int

const (
	Added Type = iota
	Updated
	Deleted
)

type Entry struct {
	changeType Type
	itemType   int
	item       any
}

func NewEntry(changeType Type, itemType int, item any) *Entry {
	return &Entry{
		changeType: changeType,
		itemType:   itemType,
		item:       item,
	}
}

func (e *Entry) GetChangeType() Type {
	return e.changeType
}

func (e *Entry) GetItemType() int {
	return e.itemType
}

func (e *Entry) GetItem() any {
	return e.item
}

type Tracker struct {
	entries []*Entry
}

func NewTracker() *Tracker {
	return &Tracker{
		entries: []*Entry{},
	}
}

func (t *Tracker) Add(entry *Entry) {
	t.entries = append(t.entries, entry)
}

func (t *Tracker) GetChanges() []*Entry {
	return t.entries
}
