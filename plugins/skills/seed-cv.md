---
name: seed-cv
description: Use when the user has an existing CV file (DOCX preferred, PDF fallback) and wants to import it into spicebag as an HTML fragment base CV and a matching CSS theme
---

Import an existing CV into spicebag. Extracts content as an HTML fragment and visual styling as a CSS theme.

**Prefer DOCX over PDF**: DOCX is structured XML with explicit styles, colors, and table data. PDF is a rendering artifact; heading detection requires heuristics and tables are often lost. If the user provides a PDF and has a DOCX available, ask them to use the DOCX instead. If only PDF is available, proceed with best-effort extraction and warn that results may be less accurate.

Required: path to a `.docx` or `.pdf` file in $ARGUMENTS.

## Process

1. Read the file at the path given in $ARGUMENTS
2. Extract CV content as an HTML fragment using the standard CV structure below: no DOCTYPE, no `<html>`, `<head>`, or `<body>` tags
3. Extract the visual theme from the document:
   - Body font family, size, line-height, and text color
   - Heading font, sizes, weights, and colors for h1/h2/h3
   - Accent color (used for headings, links, borders)
   - Page margins / max-width
   - Translate to CSS using the standard theme selectors below
4. Derive a theme name from the filename (e.g. `my-cv.docx` → `my-cv`)
5. Call `write_cv` with filename `base.html` and the HTML fragment content
6. Write the CSS theme to `~/.config/spicebag/themes/{theme-name}.css`
7. Report both paths saved

## Standard CV HTML Structure

Use this exact structure: class names must match so themes apply correctly:

```html
<h1>Full Name</h1>
<p class="cv-contact">email · phone · location · <a href="...">linkedin</a> · <a href="...">github</a></p>

<h2>Professional Summary</h2>
<p>Summary paragraph...</p>

<h2>Work Experience</h2>
<div class="cv-entry">
  <div class="cv-entry-header">
    <h3>Job Title</h3>
    <span class="cv-company">Company Name</span>
    <span class="cv-dates">Jan 2020 – Present</span>
  </div>
  <ul>
    <li>Achievement or responsibility...</li>
  </ul>
</div>

<h2>Education</h2>
<div class="cv-entry">
  <div class="cv-entry-header">
    <h3>Degree or Qualification</h3>
    <span class="cv-company">Institution Name</span>
    <span class="cv-dates">2015 – 2019</span>
  </div>
</div>

<h2>Skills</h2>
<ul>
  <li><strong>Category:</strong> item, item, item</li>
</ul>
```

## Standard CSS Theme Format

Match this selector structure: themes must target these exact elements and classes:

```css
body {
  font-family: /* extracted or closest web-safe equivalent */;
  font-size: /* e.g. 10.5pt */;
  line-height: /* e.g. 1.5 */;
  color: /* body text color */;
  max-width: /* e.g. 700px */;
  margin: 0 auto;
  padding: 40px 24px;
}

/* CV */
h1 { font-size: /* e.g. 20pt */; font-weight: 700; margin-bottom: 2px; }
.cv-contact { font-size: /* e.g. 9pt */; color: /* muted */; margin-bottom: 24px; }
h2 {
  font-size: /* e.g. 11pt */;
  font-weight: 700;
  color: /* accent */;
  border-bottom: 1px solid /* light border */;
  padding-bottom: 3px;
  margin-top: 20px;
  margin-bottom: 8px;
}
/* A4 PDF page-break control */
@page { size: A4; }
@media print { body { padding: 0; } }
.cv-entry { margin-bottom: 12px; page-break-inside: avoid; }
h2 { page-break-after: avoid; }
.cv-entry-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 4px;
}
h3 { font-size: /* e.g. 10.5pt */; font-weight: 600; margin: 0; }
.cv-company { font-weight: 600; }
.cv-dates { font-size: /* e.g. 9pt */; color: /* muted */; white-space: nowrap; }
ul { margin: 4px 0; padding-left: 18px; }
li { margin-bottom: 3px; }
p { margin: 5px 0; }
a { color: /* accent */; text-decoration: none; }

/* Cover letter */
header { margin-bottom: 28px; padding-bottom: 16px; border-bottom: 1px solid /* light border */; }
header h1 { font-size: /* e.g. 20pt */; font-weight: 600; margin-bottom: 4px; }
.contact { font-size: /* e.g. 9pt */; color: /* muted */; }
.contact a { color: /* muted */; text-decoration: none; }
.date-block { margin-bottom: 20px; font-size: /* e.g. 9.5pt */; color: /* muted */; }
.salutation { font-weight: 500; margin-bottom: 16px; }
.closing { margin-top: 28px; }
.closing .sign-off { margin-bottom: 28px; }
.closing .name { font-weight: 600; }
```

If the source document has no discernible accent color, default to `#2e75b6`.

## Rules

- Never alter factual content: names, companies, dates, skills must be verbatim
- If $ARGUMENTS is empty, ask the user for the file path before proceeding
- If the file is a PDF and a DOCX version likely exists, ask the user to provide the DOCX for better accuracy
- If the font is unavailable on the web, pick the closest web-safe or system-stack equivalent
- Save the CV as `base.html`: this is the master CV that `/customize-cv` and `/apply` build from
