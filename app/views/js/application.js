/* This file comprises javascript functions that must be assembled by the template engine. */

//getGroups loads the group template
function getGroups(prefix, div) {
  $.get('{{url "App.Groups"}}', {
    "prefix": prefix
  }, function(data) {
    $(div).html(data);
  })
}

function searchCourse(valDiv, resultDiv, dropdownID) {

  let dropdown = document.getElementById(resultDiv);
  let contains = dropdown.classList.contains("show");

  let value = $('#' + valDiv).val();
  if (value != "") {
    $.get('{{url "Course.Search"}}', {
      "value": value
    }, function(data) {
      $('#' + resultDiv).html(data);
    })

    if (!contains) {
      $('#' + dropdownID).dropdown('toggle');
    }
    dropdown.classList.remove("d-none");
  } else {

    if (contains) {
      $('#' + dropdownID).dropdown('toggle');
    }
    dropdown.classList.add("d-none");
  }
}

//on-load function to hide all elements that are not to be seen by users without the respective authority
$(function() {

  if ({{.session.role}} == "admin") {
    $(".admin").each(function() {
      $(this).removeClass("d-none");
    });
  }

  if ({{.session.role}} == "creator") {
    $(".creator").each(function() {
      $(this).removeClass("d-none");
    });
  }

  if ({{.session.isEditor}} == "true") {
    $(".editor").each(function() {
      $(this).removeClass("d-none");
    });
  }

  if ({{.session.isInstructor}} == "true") {
    $(".instructor").each(function() {
      $(this).removeClass("d-none");
    });
  }

  //detect validation error
  if ({{if .errors}}true{{else}}false{{end}}) {
    showToast($('#flash-errors').html(), 'danger');
  }
});
