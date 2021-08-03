<div id="content">
<h1>Update Password</h1>
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
    <td>Current Password:</td>
    <td><input name="current" type="password" /></td>
</tr>
<tr>
    <td>New Password:</td>
    <td><input name="password" type="password"/></td>
</tr>
<tr>
    <td>Repeat New Password:</td>
    <td><input name="password2" type="password"/></td>
</tr>
<tr><td>&nbsp;</td></tr>
<tr>
    <td>&nbsp;</td><td><input type="submit" value="Update" /></td>
</tr>
</table>
</form>
</div>
