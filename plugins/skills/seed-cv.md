---
name: seed-cv
description: Use when the user has an existing CV file (DOCX preferred, PDF fallback) and wants to import it into spicebag as a markdown base CV and a matching CSS theme
---

Import an existing CV into spicebag. Extracts content as markdown and visual styling as a CSS theme.

**Prefer DOCX over PDF** — DOCX is structured XML with explicit styles, colors, and table data. PDF is a rendering artifact; heading detection requires heuristics and tables are often lost. If the user provides a PDF and has a DOCX available, ask them to use the DOCX instead. If only PDF is available, proceed with best-effort extraction and warn that results may be less accurate.

Required: path to a `.docx` or `.pdf` file in $ARGUMENTS.

## Process

1. Read the DOCX file at the path given in $ARGUMENTS
2. Extract CV content — convert to clean markdown:
   - H1 for the person's name
   - H2 for section headings (Experience, Education, Skills, etc.)
   - H3 for job titles / company names
   - Bullet points for responsibilities and achievements
   - Keep all factual data exactly as written
3. Extract the visual theme from the DOCX:
   - Body font family, size, line-height, and text color
   - Heading font, sizes, weights, and colors for H1/H2/H3
   - Accent color (used for headings, links, borders)
   - Page margins / max-width
   - List spacing
   - Translate to CSS in the same structure as spicebag's built-in themes (see below)
4. Derive a theme name from the filename (e.g. `my-cv.docx` → `my-cv`, `my-cv.pdf` → `my-cv`)
5. Call `write_cv` with filename `base.md` and the extracted markdown content
6. Write the CSS theme to `~/.config/spicebag/themes/{theme-name}.css`
7. Report both paths saved

## CSS Theme Format

Match this structure exactly — spicebag renders markdown CVs with these selectors:

```css
body {
  font-family: /* extracted or closest web-safe equivalent */;
  font-size: /* e.g. 10.5pt */;
  line-height: /* e.g. 1.5 */;
  color: /* body text color */;
  max-width: /* e.g. 780px */;
  margin: 0 auto;
  padding: 40px 24px;
}
h1 { font-size: /* e.g. 24pt */; font-weight: 700; color: /* accent */; margin-bottom: 4px; }
h2 {
  font-size: /* e.g. 11pt */;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: /* accent */;
  border-bottom: 1px solid /* light border color */;
  padding-bottom: 4px;
  margin-top: 22px;
}
h3 { font-size: /* e.g. 10.5pt */; font-weight: 600; margin-bottom: 2px; }
ul { margin: 4px 0; padding-left: 18px; }
li { margin-bottom: 3px; }
p { margin: 5px 0; }
a { color: /* accent */; text-decoration: none; }
```

If the DOCX has no discernible accent color, default to `#1a56db`.

## Rules

- Never alter factual content — names, companies, dates, skills must be verbatim
- If $ARGUMENTS is empty, ask the user for the file path before proceeding
- If the file is a PDF and a DOCX version exists, ask the user to provide the DOCX for better accuracy
- If the font is unavailable on the web, pick the closest web-safe or system-stack equivalent
- Save the CV as `base.md` — this is the master CV that `/customize-cv` and `/apply` build from
