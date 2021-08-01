<div id="content">
&nbsp;
{{if .flash.error}}
<h3>{{.flash.error}}</h3>
&nbsp;
{{end}}
{{if .Errors}}
{{range $rec := .Errors}}
<h3>{{$rec}}</h3>
{{end}}
&nbsp;
{{end}}
{{if .Cancelled}}
<h2>Your account has <b>NOT</b> been cancelled, yet.</h2>
{{else}}
<h2>Your account is deactivated. Your account will be deactivated after your next logout.</h2>
{{end}}
</div>
