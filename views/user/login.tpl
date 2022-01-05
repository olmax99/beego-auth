<div id="content">
    <div class="container">
        <div id=message-row class="row">
            <div id="message-box" class="column">
                <div  class="card" style="width: 18rem;">
                    <div class="card-body">
                        <h4 class="card-title">Welcome</h5>
                            &nbsp;
                            {{if .flash.error}}
                            <p>{{.flash.error}}</p>
                            &nbsp;
                            {{end}}
                            {{if .Errors}}
                            {{range $rec := .Errors}}
                            <p>{{$rec}}</p>
                            {{end}}
                            &nbsp;
                            {{end}}
                            <div id="message-content" class="card-text"></div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    <form method="POST">
        <table>
            <tr>
                <td>Email address:</td>
                <td><input name="email" type="text" autofocus /></td>
            </tr>
            <tr>
                <td>Password:</td>
                <td><input name="password" type="password" /></td>
            </tr>
            <tr>
                <td>&nbsp;</td>
            </tr>
            <tr>
                <td>&nbsp;</td>
                <td><input type="submit" value="Login" /></td>
                <td><a href="http://localhost:{{.Httpport}}/user/register">Register</a></td>
                <td><a href="http://localhost:{{.Httpport}}/user/reset">Reset Password</a></td>
            </tr>
        </table>
    </form>
</div>
