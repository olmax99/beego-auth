<div id="content">
<h1>Remove User Account</h1>
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
<p><font size="3">Provide your password for proceeding with the cancellation of your account.</font></p>
<form method="POST">
    <table>
        <tr>
            <td>Current password:</td>
            <td><input name="current" type="password" /></td>
        </tr>
        <tr>
            <td>&nbsp;</td>
        </tr>
        <tr>
            <td>&nbsp;</td>
            <td><input type="submit" value="Remove" /></td>
            <td><a href="http://localhost:{{.Httpport}}/home">Cancel</a></td>
        </tr>
    </table>
</form>
</div>
