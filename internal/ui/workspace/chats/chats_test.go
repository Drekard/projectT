package chats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChatsUI(t *testing.T) {
	ui := New()
	
	assert.NotNil(t, ui)
	// Note: We can't test ui.content directly in unit tests due to Fyne requirements
}

func TestChatsUICreateView(t *testing.T) {
	ui := New()
	
	view := ui.CreateView()
	
	assert.NotNil(t, view)
	// Note: We can't fully test the view in unit tests due to Fyne requirements
}

func TestChatsUIGetContent(t *testing.T) {
	ui := New()
	
	content := ui.GetContent()
	
	assert.NotNil(t, content)
	// Note: We can't compare ui.content directly in unit tests due to Fyne requirements
}