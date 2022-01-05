<!DOCTYPE html>

<html>

<head>
    <meta http-equiv="content-type" content="text/html; charset=utf-8" />
    <link rel="shortcut icon" href="/static/img/favicon.png" />
    <meta name="author" content="Unknown" />
    <meta name="description" content="Beego authentication" />
    <meta name="keywords" content="Go, golang, beego, authentication">

    <title>Astronomy Website Template</title>

    <link href="/static/css/default.css" rel="stylesheet" type="text/css" />
    <link href="/static/css/bootstrap.min.css" rel="stylesheet" />

    <script src="/static/js/jquery-1.10.1.min.js"></script>

    <script src="/static/js/notifier.js"></script>
</head>

<body>
    <!-- start header -->
    <div id="header">
        <div align="right">
            {{if .InSession}}
            Welcome, {{.First}} [<a href="http://localhost:{{.Httpport}}/user/logout">Logout</a>|<a href="http://localhost:{{.Httpport}}/user/profile">Profile</a>]
            {{else}}
            [<a href="http://localhost:{{.Httpport}}/user/login/console">Login</a>]
            {{end}}
        </div>
        <div class="wrapper clearfix">
            <div id="logo">
                <a href="#"><img src="/static/img/logo.png" alt="LOGO"></a>
            </div>
            <ul id="navigation">
                {{if .InSession}}
                <li class="selected">
                    <a href="http://localhost:{{.Httpport}}/console">home</a>
                </li>
                {{end}}
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
