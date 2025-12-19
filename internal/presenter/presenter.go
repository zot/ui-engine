// Package presenter implements the Presenter System.
// CRC: crc-Presenter.md, crc-AppPresenter.md, crc-ListPresenter.md
// Spec: main.md
package presenter

import (
	"encoding/json"
)

// Presenter represents a UI presenter.
type Presenter struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data,omitempty"`
	ViewName   string          `json:"view,omitempty"`
	VariableID int64           `json:"-"`
}

// NewPresenter creates a new presenter.
func NewPresenter(presenterType string) *Presenter {
	return &Presenter{
		Type:     presenterType,
		ViewName: "DEFAULT",
	}
}

// GetData returns the presenter state.
func (p *Presenter) GetData() json.RawMessage {
	return p.Data
}

// SetData updates the presenter state.
func (p *Presenter) SetData(data json.RawMessage) {
	p.Data = data
}

// GetType returns the presenter type.
func (p *Presenter) GetType() string {
	return p.Type
}

// GetViewName returns the active view name.
func (p *Presenter) GetViewName() string {
	if p.ViewName == "" {
		return "DEFAULT"
	}
	return p.ViewName
}

// SetViewName switches to a different view.
func (p *Presenter) SetViewName(name string) {
	p.ViewName = name
}

// AppPresenter is the root presenter for a session.
type AppPresenter struct {
	*Presenter
	URL          string          `json:"url"`
	HistoryIndex int             `json:"historyIndex"`
	History      []PageReference `json:"history"`
}

// PageReference refers to a page presenter.
type PageReference struct {
	VariableID int64  `json:"variableId"`
	URL        string `json:"url"`
}

// NewAppPresenter creates a new app presenter.
func NewAppPresenter() *AppPresenter {
	return &AppPresenter{
		Presenter:    NewPresenter("app"),
		URL:          "/",
		HistoryIndex: 0,
		History:      []PageReference{},
	}
}

// CurrentPage returns the current page from history.
func (ap *AppPresenter) CurrentPage() *PageReference {
	if ap.HistoryIndex < 0 || ap.HistoryIndex >= len(ap.History) {
		return nil
	}
	return &ap.History[ap.HistoryIndex]
}

// Navigate pushes a new page to history.
func (ap *AppPresenter) Navigate(url string, variableID int64) {
	ap.URL = url

	// Truncate forward history
	if ap.HistoryIndex < len(ap.History)-1 {
		ap.History = ap.History[:ap.HistoryIndex+1]
	}

	// Push new page
	ap.History = append(ap.History, PageReference{
		VariableID: variableID,
		URL:        url,
	})
	ap.HistoryIndex = len(ap.History) - 1
}

// Back navigates to the previous page.
func (ap *AppPresenter) Back() bool {
	if ap.HistoryIndex > 0 {
		ap.HistoryIndex--
		if page := ap.CurrentPage(); page != nil {
			ap.URL = page.URL
		}
		return true
	}
	return false
}

// Forward navigates to the next page.
func (ap *AppPresenter) Forward() bool {
	if ap.HistoryIndex < len(ap.History)-1 {
		ap.HistoryIndex++
		if page := ap.CurrentPage(); page != nil {
			ap.URL = page.URL
		}
		return true
	}
	return false
}

// Go navigates to a specific history index.
func (ap *AppPresenter) Go(index int) bool {
	if index >= 0 && index < len(ap.History) {
		ap.HistoryIndex = index
		if page := ap.CurrentPage(); page != nil {
			ap.URL = page.URL
		}
		return true
	}
	return false
}

// ReplaceCurrentPage replaces the current page without adding to history.
func (ap *AppPresenter) ReplaceCurrentPage(url string, variableID int64) {
	ap.URL = url
	if ap.HistoryIndex >= 0 && ap.HistoryIndex < len(ap.History) {
		ap.History[ap.HistoryIndex] = PageReference{
			VariableID: variableID,
			URL:        url,
		}
	} else {
		ap.History = []PageReference{{VariableID: variableID, URL: url}}
		ap.HistoryIndex = 0
	}
}

// ToData serializes the app presenter state.
func (ap *AppPresenter) ToData() (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{
		"url":          ap.URL,
		"historyIndex": ap.HistoryIndex,
		"history":      ap.History,
	})
}

// ListPresenter manages a list of items.
type ListPresenter struct {
	*Presenter
	Items      []int64 `json:"items"` // Variable IDs of list items
	SelectedID int64   `json:"selectedId,omitempty"`
}

// NewListPresenter creates a new list presenter.
func NewListPresenter() *ListPresenter {
	return &ListPresenter{
		Presenter: NewPresenter("list"),
		Items:     []int64{},
	}
}

// AddItem adds an item to the list.
func (lp *ListPresenter) AddItem(variableID int64) {
	lp.Items = append(lp.Items, variableID)
}

// RemoveItem removes an item from the list.
func (lp *ListPresenter) RemoveItem(variableID int64) bool {
	for i, id := range lp.Items {
		if id == variableID {
			lp.Items = append(lp.Items[:i], lp.Items[i+1:]...)
			if lp.SelectedID == variableID {
				lp.SelectedID = 0
			}
			return true
		}
	}
	return false
}

// SelectItem selects an item.
func (lp *ListPresenter) SelectItem(variableID int64) {
	lp.SelectedID = variableID
}

// GetSelected returns the selected item ID.
func (lp *ListPresenter) GetSelected() int64 {
	return lp.SelectedID
}

// GetItems returns all item IDs.
func (lp *ListPresenter) GetItems() []int64 {
	result := make([]int64, len(lp.Items))
	copy(result, lp.Items)
	return result
}

// ToData serializes the list presenter state.
func (lp *ListPresenter) ToData() (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{
		"items":      lp.Items,
		"selectedId": lp.SelectedID,
	})
}
