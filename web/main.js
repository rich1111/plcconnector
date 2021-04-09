document.addEventListener("DOMContentLoaded", function (e) {
  var b = document.getElementById("showbtn");
  b.addEventListener("click", function (e) {
    document.documentElement.style.setProperty('--hide', b.checked ? 'table-row' : 'none');
  });
});
