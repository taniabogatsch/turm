/* This file comprises functios for managing participants of a course. */

function toggleEventSelection(elemID) {

  let selected = $('#' + elemID).val();

  let select = document.getElementById(elemID + "-options");
  if (selected == "false") {
    select.classList.remove("d-none");
  } else {
    select.classList.add("d-none");
  }
}

function submitParticipantsModal(elemID) {
  $('#' + elemID + '-form').submit();
  $('#' + elemID + '-modal').modal('hide');
}
