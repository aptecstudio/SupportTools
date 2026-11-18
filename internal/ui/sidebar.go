// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

package ui

import (
	"github.com/lxn/walk"

	"github.com/aptecstudio/supporttools/internal/modules"
)

// groupItem è un nodo di primo livello della barra laterale: un gruppo di tool.
type groupItem struct {
	group    modules.Group
	children []*moduleItem
}

func (g *groupItem) Text() string                { return g.group.Title() }
func (g *groupItem) Parent() walk.TreeItem       { return nil }
func (g *groupItem) ChildCount() int             { return len(g.children) }
func (g *groupItem) ChildAt(i int) walk.TreeItem { return g.children[i] }

// moduleItem è un tool nella barra laterale. page è creata alla prima
// apertura e poi riutilizzata, così il tool conserva il proprio stato.
type moduleItem struct {
	parent *groupItem
	module modules.Module
	page   *walk.Composite
}

func (m *moduleItem) Text() string              { return m.module.Title }
func (m *moduleItem) Parent() walk.TreeItem     { return m.parent }
func (m *moduleItem) ChildCount() int           { return 0 }
func (m *moduleItem) ChildAt(int) walk.TreeItem { return nil }

// sidebarModel è il modello ad albero della barra laterale.
type sidebarModel struct {
	walk.TreeModelBase
	roots []*groupItem
}

// newSidebarModel raggruppa i tool (già ordinati da modules.All) per gruppo.
func newSidebarModel(mods []modules.Module) *sidebarModel {
	m := new(sidebarModel)
	var current *groupItem
	for _, mod := range mods {
		if current == nil || current.group != mod.Group {
			current = &groupItem{group: mod.Group}
			m.roots = append(m.roots, current)
		}
		current.children = append(current.children, &moduleItem{parent: current, module: mod})
	}
	return m
}

func (m *sidebarModel) RootCount() int             { return len(m.roots) }
func (m *sidebarModel) RootAt(i int) walk.TreeItem { return m.roots[i] }
