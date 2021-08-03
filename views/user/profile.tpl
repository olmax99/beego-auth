<div id="content">
<h1>Your Profile</h1>
&nbsp;
{{if .flash.error}}
<h3>{{.flash.error}}</h3>
&nbsp;
{{end}}{{if .flash.notice}}
<h3>{{.flash.notice}}</h3>
&nbsp;
{{end}}
{{if .Errors}}
{{range $rec := .Errors}}
<h3>{{$rec}}</h3>
{{end}}
&nbsp;
{{end}}
<form method="POST">
<table>
<tr>
    <td>First name:</td>
    <td><input name="first" type="text" value="{{.First}}" /></td>
</tr>
<tr>
    <td>Last name:</td>
    <td><input name="last" type="text" value="{{.Last}}"/></td>
</tr>
<tr>
    <td>Email address: (Cannot be changed)</td>
    <td><input name="email" type="text" value="{{.Email}}"/></td>
</tr>
<tr>
    <td>Current password:</td>
    <td><input name="current" type="password" /></td>
</tr>
<tr><td>&nbsp;</td></tr>
<tr>
    <td>&nbsp;</td><td><input type="submit" value="Update" /></td>
</tr>
</table>
<a href="http://localhost:{{.Httpport}}/user/remove">Remove account</a>
</form>
</div>
