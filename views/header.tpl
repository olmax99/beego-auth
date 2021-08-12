<!DOCTYPE html>

<html>

<head>
    <meta http-equiv="content-type" content="text/html; charset=utf-8" />
    <title>Astronomy Website Template</title>
    <link href="/static/css/default.css" rel="stylesheet" type="text/css" />
</head>

<body>
    <!-- start header -->
    <div id="header">
        <div align="right">
            {{if .InSession}}
            Welcome, {{.First}} [<a href="http://localhost:{{.Httpport}}/user/logout">Logout</a>|<a href="http://localhost:{{.Httpport}}/user/profile">Profile</a>]
            {{else}}
                [<a href="http://localhost:{{.Httpport}}/user/login/home">Login</a>]
                {{end}}
        </div>
        <div class="wrapper clearfix">
            <div id="logo">
                <a href="#"><img src="/static/img/logo.png" alt="LOGO"></a>
            </div>
            <ul id="navigation">
                <li class="selected">
                    <a href="http://localhost:{{.Httpport}}/home">home</a>
                </li>
                <li>
                    <a href="#">About</a>
                </li>
                <li>
                    <a href="#">Blog</a>
                </li>
                <li>
                    <a href="#">Gallery</a>
                </li>
                <li>
                    <a href="#">Contact Us</a>
                </li>
            </ul>
        </div>
    </div>
    <!-- end header -->
    <!-- start page -->
    <div id="page">
