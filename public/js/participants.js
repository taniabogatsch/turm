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

function reactToEntryInput(eventIdx, courseID, eventID) {

  document.getElementById("search-form-" + eventIdx).classList.add('was-validated');
  const value = $('#user-search-input-' + eventIdx).val();

  if (value.length > 2) { //search matching users
    searchEntries(eventIdx, courseID, eventID, value);

  } else if (value.length == 0) {
    $('#user-search-results-' + eventIdx).html("");
    document.getElementById("search-form-" + eventIdx).classList.remove('was-validated');

  } else {
    $('#user-search-results-' + eventIdx).html("");
  }
}

function openChangeStatusModal(eventID, userID, status) {

  $('#change-status-event-ID').val(eventID);
  $('#change-status-user-ID').val(userID);

  if (status == "awaiting payment") {
    $('#change-status-select').val("2");
  } else if (status == "paid") {
    $('#change-status-select').val("3");
  } else {
    $('#change-status-select').val("4");
  }

  $('#change-status-modal').modal('show');
}

function renderDays(ID, eventID, shift, monday, action) {

  $.get(action, {
    "ID": ID,
    "eventID": eventID,
    "shift": shift,
    "t": monday
  }, function(data) {
    $('#participants-slots-days-' + eventID).html(data);
  })
}

function deleteSlot(ID, eventID, slotID, monday, action) {

  $.get(action, {
    "ID": ID,
    "eventID": eventID,
    "slotID": slotID,
    "t": monday
  }, function(data) {
    $('#participants-slots-days-' + eventID).html(data);
  })
}
