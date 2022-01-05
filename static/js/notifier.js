var lastReceived = 0;
var isWait = false;

var fetch = function () {
  if (isWait) return;
  isWait = true;
  $.getJSON("/notifier/fetch/" + lastReceived, function (data) {
    if (data == null) return;
    $.each(data, function (i, event) {
      var elem = document.createElement('div');

      switch (event.Type) {
      case 0:
        var time = document.createElement('span');
        var username = document.createElement('strong');
        var content = document.createElement('span');

        time.innerText = event.Timeread;
        username.innerText = event.User;
        content.innerText = event.Content;

        elem.appendChild(document.createTextNode('['));
        elem.appendChild(time);
        elem.appendChild(document.createTextNode(']'));
        elem.appendChild(username);
        elem.appendChild(document.createTextNode(': '));
        elem.appendChild(content);
        // TODO: add closing class
        // <a href="#"><span class="close-message">close</span></a>

        // if (!(time.innerText.indexOf('undefined') === 0)) {
        //   $('#message-content').append(elem);
        // }
        $('#message-content').append(elem);

        break;
      }

      $('#message-content').fadeIn(1000, function() {
        $('#message-content').delay(10000).fadeOut(1000)
      });

      lastReceived = event.Timestamp;
    });
    isWait = false;
  });
}

setInterval(fetch, 2000);

fetch();

$(document).ready(function () {
  $(".close-message").click(function() {
    $("#message-content").stop(true).fadeOut("slow");
  });
});
