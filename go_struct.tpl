package model

import (
    "time"
)


{{- range $ti, $t  := .Types }}
  {{- if not $t.BuiltIn }}
    {{- if eq $t.Kind "OBJECT" }}
      {{- if and (ne $t.Name "Query") (ne $t.Name "Mutation") (ne $t.Name "Subscription") }}
type {{ $t.Name | title }} struct {
    {{ if not (foundPK $t.Name $t.Fields) }}{{ joinstr (camelcase $t.Name) "Id" }}   string `spanner:"{{ untitle (joinstr $t.Name "Id") }}"`{{ end }}
        {{- range $fi, $f  := $t.Fields }}
          {{- $cfn := ConvertObjectFieldName $f }}
    {{ $f.Name | title }}   {{ GoType $f false }} `{{ if (eq $cfn $f.Name) }}spanner:"{{ $f.Name }}" {{ end }}json:"{{ $f.Name }}"`
          {{- if not (eq $cfn $f.Name) }}
    {{ $cfn | title }}   {{ GoType $f true }} `spanner:"{{ $cfn }}"`
          {{- end }}
        {{- end }}
    {{ if not (exists $t "CreatedAt") }}CreatedAt   time.Time `spanner:"createdAt"`{{ end }}
    {{ if not (exists $t "UpdatedAt") }}UpdatedAt   time.Time `spanner:"updatedAt"`{{ end }}
}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
