/* This file comprises javascript functions are independent of the template engine. */

//on-load function for disabling form submissions if there are invalid fields
(function() {
  'use strict';
  window.addEventListener('load', function() {
    //fetch all the forms to which we want to apply custom bootstrap validation styles
    var forms = document.getElementsByClassName('needs-validation');
    //loop over them and prevent submission
    var validation = Array.prototype.filter.call(forms, function(form) {
      form.addEventListener('submit', function(event) {
        if (form.checkValidity() === false) {
          event.preventDefault();
          event.stopPropagation();
        }
        form.classList.add('was-validated');
      }, false);
    });
  }, false);
})();

//changeIcon adjusts the icons (by id) according to the collapse state
function changeIcon(id) {

  //get icons
  const iconRightClass = $("#icon-right-" + id).attr("class");
  const iconDownClass = $("#icon-down-" + id).attr("class");

  if (iconRightClass == "display-block") {
    $("#icon-right-" + id).attr("class", "display-none");
    $("#icon-down-" + id).attr("class", "display-block");
  } else {
    $("#icon-right-" + id).attr("class", "display-block");
    $("#icon-down-" + id).attr("class", "display-none");
  }
}

//showErrorToast loads an error message into the toast and shows it
function showErrorToast(content) {

  $("#toast-title").html($("#icon-flash-alertCircle").html());
  $("#toast-title").attr("class", "mr-auto color-danger");
  $("#toast-content").html(content);
  $("#toast-content").attr("class", "color-danger");
  $("#feedback-toast").toast('show');
}

//showSuccessToast loads a success message into the toast and shows it
function showSuccessToast(content) {

  $("#toast-title").html($("#icon-flash-check").html());
  $("#toast-title").attr("class", "mr-auto color-success");
  $("#toast-content").html(content);
  $("#toast-content").attr("class", "color-success");
  $("#feedback-toast").toast('show');
}

//openGroupsNav shows the groups modal of the navigation bar
function openGroupsNav() {
  getGroups("nav", '#nav-groups-modal-content');
  $('#nav-groups-modal').modal('show');
}
