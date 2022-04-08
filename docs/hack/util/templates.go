package util

const PagePathFunction = "docs/pages/configuration/functions/%s.mdx"
const PartialPathArgs = "docs/pages/configuration/functions/_partials/%s/_args.mdx"
const PartialPathFlag = "docs/pages/configuration/functions/_partials/%s/%s.mdx"
const TemplatePage = `---
title: Function %s
sidebar_label: %s
---
%s

%s

## Arguments
%s


## Flags

%s
`
const TemplatePartialImport = `
import %s from "%s"`
const TemplatePartialUse = `<%s />
`
const TemplatePartialUseConfig = `<%s />
`
const TemplatePartialUseFunction = `<%s function="%s" />
`
const TemplatePartialUseFlag = `<%s function="%s" flag="%s" />
`
const AutoGenTagBegin = "<!--- BEGIN AUTO GENREATED CONTENT -->"
const AutoGenTagEnd = "<!--- END AUTO GENREATED CONTENT -->"
const TemplateFlag = AutoGenTagBegin + `
### ` + "`--%s%s`" + `
%s
` + AutoGenTagEnd + "\n\n"
const TemplateConfigField = `
<details className="config-field" data-expandable="%t"%s>
<summary>

%s` + "`%s`" + ` <span class="config-field-required" data-required="%t">required</span> <span class="config-field-type">%s</span> <span class="config-field-default">%s</span> <span class="config-field-enum">%s</span>

%s

</summary>

%s

</details>
`
