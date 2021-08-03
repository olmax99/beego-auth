<div id="content">
<h1>Reset User Password</h1>
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
<p><font size="3">Provide your email for proceeding with resetting your password.</font></p>
<form method="POST">
    <table>
         <tr>      
             <td>Email:</td>
             <td><input name="current" type="email" /></td>
         </tr>
         <tr>
             <td>&nbsp;</td>
         </tr>
         <tr>
             <td>&nbsp;</td>
             <td><input type="submit" value="Reset" /></td>
             <td><a href="http://localhost:{{.Httpport}}/home">Cancel</a></td>
         </tr>
    </table>
</form>
</div>
