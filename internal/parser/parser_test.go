// internal/parser/parser_test.go
package parser_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cvWithFrontmatter = `---
experience:
  - role_type: backend
    company: "Acme Corp"
    start: "2018-01-01"
    end: "2020-06-01"
  - role_type: devops
    company: "FooCo"
    start: "2022-03-01"
    end: ""
---

# John Doe
Senior Engineer
`

func TestParseExperience(t *testing.T) {
	entries, err := parser.ParseExperience(cvWithFrontmatter)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "backend", entries[0].RoleType)
	assert.Equal(t, "Acme Corp", entries[0].Company)
	assert.Equal(t, "2018-01-01", entries[0].StartDate)
	assert.Equal(t, "2020-06-01", entries[0].EndDate)

	assert.Equal(t, "devops", entries[1].RoleType)
	assert.Equal(t, "", entries[1].EndDate) // ongoing
}

func TestParseExperienceNoFrontmatter(t *testing.T) {
	entries, err := parser.ParseExperience("# Just a CV\nNo frontmatter here")
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}
