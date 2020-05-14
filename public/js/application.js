/* This file comprises javascript functions are independent of the template engine. */

//on-load function for disabling form submissions if there are invalid fields
(function() {
  'use strict';
  window.addEventListener('load', function() {
    //fetch all the forms to which we want to apply custom bootstrap validation styles
    let forms = document.getElementsByClassName('needs-validation');
    //loop over them and prevent submission
    let validation = Array.prototype.filter.call(forms, function(form) {
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

  if (iconRightClass == "d-block") {
    $("#icon-right-" + id).attr("class", "d-none");
    $("#icon-down-" + id).attr("class", "d-block");
  } else {
    $("#icon-right-" + id).attr("class", "d-block");
    $("#icon-down-" + id).attr("class", "d-none");
  }
}

//showErrorToast loads an error message into the toast and shows it
function showErrorToast(content) {

  $("#toast-title").html($("#icon-flash-alertCircle").html());
  $("#toast-title").attr("class", "mr-auto text-danger");
  $("#toast-content").html(content);
  $("#toast-content").attr("class", "text-danger");
  $("#feedback-toast").toast('show');
}

//showSuccessToast loads a success message into the toast and shows it
function showSuccessToast(content) {

  $("#toast-title").html($("#icon-flash-check").html());
  $("#toast-title").attr("class", "mr-auto text-success");
  $("#toast-content").html(content);
  $("#toast-content").attr("class", "text-success");
  $("#feedback-toast").toast('show');
}

//confirmPOSTModal confirms the execution of a POST action
function confirmPOSTModal(title, content, action) {

  $('#confirm-POST-modal-title').html(title);
  $('#confirm-POST-modal-form').attr("action", action);
  $('#confirm-POST-modal-content').html(content);

  //show the modal
  $('#confirm-POST-modal').modal('show');
}

//openGroupsNav shows the groups modal of the navigation bar
function openGroupsNav() {
  getGroups("nav", '#nav-groups-modal-content');
  $('#nav-groups-modal').modal('show');
}
