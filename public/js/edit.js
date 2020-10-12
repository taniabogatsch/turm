/* This file comprises js functions required to load the different modals for editing a course. */

function submitChangeGroupModal(parentID) {

  $('#change-group-modal-parentID').val(parentID);
  $('#change-group-modal-form').submit();
}

function openChangeModal(title, field, valid, action, modal, max, info, ID) {

  let value = $('#div-' + field).html();
  value = value.trim();

  $('#change-' + modal + '-modal-ID').val(ID);
  $('#change-' + modal + '-modal-title').html(title);
  $('#change-' + modal + '-modal-field').val(field);
  $('#change-' + modal + '-modal-form').attr('action', action);
  $('#change-' + modal + '-modal-info').html(info);

  //set the value
  if (valid) {
    if (modal == "timestamp") {
      const timestampParts = value.split(" ");
      $('#change-timestamp-modal-date').val(timestampParts[0]);
      $('#change-timestamp-modal-time').val(timestampParts[1]);
    } else {
      $('#change-' + modal + '-modal-value').val(value);
    }

  } else {
    if (modal == "timestamp") {
      $('#change-timestamp-modal-date').val("");
      $('#change-timestamp-modal-time').val("");
    } else {
      $('#change-' + modal + '-modal-value').val("");
      if (field == "event") {
        title = value;
      }
      $('#change-' + modal + '-modal-value').attr("placeholder", title);
    }
  }

  //set max string length for text inputs
  if (modal == "text" && field != "fee") {
    $('#change-text-modal-value').attr("maxlength", max);
  } else {
    $('#change-text-modal-value').removeAttr("maxlength");
  }

  if (field == "fee") {
    $('#change-text-modal-icon').html($('#change-text-icon-euro').html());
    $('#change-text-modal-value').attr("pattern", "^([0-9]{1,6}([,|.][0-9]{0,2})?)?");
    $('#change-text-modal-validation').html($('#change-text-validation-fee').html());
    $('#change-text-modal-value').attr("maxlength", 10);
  } else {
    $('#change-text-modal-icon').html($('#change-text-icon-pencil').html());
    $('#change-text-modal-value').removeAttr("pattern");
    $('#change-text-modal-validation').html($('#change-text-validation-text').html());
  }

  if (field == "event") {
    $('#change-text-modal-btn').html($('#change-text-add').html());
  } else {
    $('#change-text-modal-btn').html($('#change-text-save').html());
  }

  //show the modal
  $('#change-' + modal + '-modal').modal('show');
}

function openUserListModal(title, listType) {

  //reset the search
  $('#change-user-list-modal-search').val("");
  reactToListInput();

  $('#change-user-list-modal-title').html(title);
  $('#change-user-list-modal-list').val(listType);

  //show the modal
  $('#change-user-list-modal').modal('show');
}

function reactToListInput() {

  //get the list type
  const listType = $('#change-user-list-modal-list').val();

  //validate the form
  document.getElementById("change-user-list-modal-form").classList.add('was-validated');

  //get the search value
  const value = $('#change-user-list-modal-search').val();

  if (value.length > 2) { //search matching users
    const searchInactive = $('#change-user-list-modal-checkbox').is(':checked');
    const courseID = $('#change-user-list-modal-courseID').val();
    searchForList(value, searchInactive, listType, courseID);

  } else if (value.length == 0) { //no search value entered
    $('#change-user-list-modal-results').html("");
    document.getElementById("change-user-list-modal-form").classList.remove('was-validated');

  } else { //not enough characters
    $('#change-user-list-modal-results').html("");
  }
}

function submitList(ID) {
  //set the user ID and submit
  $('#change-user-list-modal-user').attr("value", ID);
  $('#change-user-list-modal-form').submit();
}

function openBoolModal(title, action, option1, option2, value, userID, listType, ID) {

  $('#change-bool-modal-ID').val(ID);
  $('#change-bool-modal-title').html(title);
  $('#change-bool-modal-form').attr("action", action);

  //set options and show the correct one
  $('#change-bool-modal-option-1').html(option1);
  $('#change-bool-modal-option-2').html(option2);
  $("#change-bool-modal-checkbox").prop("checked", value);
  if (value) {
    $('#change-bool-modal-option-1').removeClass("d-none");
    $('#change-bool-modal-option-2').addClass("d-none");
    $('#change-bool-modal-option-1').addClass("d-inline");
    $('#change-bool-modal-option-2').removeClass("d-inline");
  } else {
    $('#change-bool-modal-option-1').addClass("d-none");
    $('#change-bool-modal-option-2').removeClass("d-none");
    $('#change-bool-modal-option-1').removeClass("d-inline");
    $('#change-bool-modal-option-2').addClass("d-inline");
  }

  //set optional values: userID and listType
  $('#change-bool-modal-user').val(userID);
  $('#change-bool-modal-list').val(listType);

  //show the modal
  $('#change-bool-modal').modal('show');
}

function openTextAreaModal(title, field, valid, action, info, isEMail) {

  $('#change-text-area-modal-title').html(title);
  $('#change-text-area-modal-field').val(field);
  $('#change-text-area-modal-form').attr("action", action);
  $('#change-text-area-modal-info').html(info);

  let fields = document.getElementsByClassName("only-custom-email");
  for (let i = 0; i < fields.length; i++) {
    if (isEMail) {
      fields[i].classList.remove('d-none');
    } else {
      fields[i].classList.add('d-none');
    }
  }

  //set content
  if (valid) {
    quill.root.innerHTML = $('#div-' + field).html();
  } else {
    quill.root.innerHTML = "";
  }

  //show the modal
  $('#change-text-area-modal').modal('show');
}

function submitTextArea() {

  document.getElementById("change-text-area-modal-form").classList.add('was-validated');
  const textArea = document.getElementById("change-text-area-modal-value");

  const text = quill.root.innerHTML;

  if (text != "<p><br></p>") {
    $('#change-text-area-modal-value').val(text);
    textArea.setCustomValidity('');
    $('#change-text-area-modal-form').submit();
  } else {
    textArea.setCustomValidity("Please provide a text.");
  }
}

function openNewMeetingModal(eventID) {

  $('#new-meeting-modal-ID').val(eventID);
  $('#new-meeting-modal').modal('show');
}

function openEditMeeting(meetingID, start, end, place, annotation, weekday, interval) {

  let meetingType = "single";
  if (interval != 0) {
    meetingType = "weekly";
    //$('#meeting-weekly-interval').val(interval); //TODO
    //$('#meeting-weekly-weekday').val(weekday); //TODO
  }

  $('#edit-meeting-' + meetingType + '-ID').val(meetingID);

  if (start != "") {
    const startParts = start.split(" ");
    $('#' + meetingType + '-start-date').val(startParts[0]);
    $('#' + meetingType + '-start-time').val(startParts[1]);
  }
  if (end != "") {
    const endParts = end.split(" ");
    $('#' + meetingType + '-end-date').val(endParts[0]);
    $('#' + meetingType + '-end-time').val(endParts[1]);
  }

  $('#meeting-' + meetingType + '-place').val(place);
  $('#meeting-' + meetingType + '-annotation').val(annotation);

  $('#edit-meeting-' + meetingType).modal('show');
}

function plainCourse() {
  $(".edit-show").each(function() {
    $(this).addClass("d-none");
  });
  $(".edit-hide").each(function() {
    $(this).removeClass("d-none");
  });
  $('#preview-btn').addClass('d-none');
  $('#hide-preview-btn').removeClass('d-none');
}

function editCourse() {
  $(".edit-show").each(function() {
    $(this).removeClass("d-none");
  });
  $(".edit-hide").each(function() {
    $(this).addClass("d-none");
  });
  $('#preview-btn').removeClass('d-none');
  $('#hide-preview-btn').addClass('d-none');
}

function openRestrictionModal(title, ID, degreeID, studiesID, minSemester) {

  $('#change-restriction-modal-title').html(title);
  $('#change-restriction-modal-restriction-ID').val(ID);
  $('#change-restriction-modal-select-degree').val(degreeID);
  $('#change-restriction-modal-select-studies').val(studiesID);

  if (minSemester != 0) {
    $('#change-restriction-modal-minimum-semester').val(minSemester);
  } else {
    $('#change-restriction-modal-minimum-semester').val('');
  }

  //show the modal
  $('#change-restriction-modal').modal('show');
}

function openEnrollmentKeyModal(eventID) {

  $('#change-enrollment-key-event-ID').val(eventID);
  $('#change-enrollment-key-modal').modal('show');
}

function handleEditResult(response) {

  if (response.FieldID == "title") {
    $('#div-' + response.FieldID).html(response.Value);

  } else  if (response.FieldID == "subtitle" || response.FieldID == "fee" ||
    response.FieldID == "speaker" || response.FieldID == "description" ||
    response.FieldID == "custom_email") {

    if (response.Value != "") {
      document.getElementById("div-edit-" + response.FieldID).classList.remove("d-none");
      document.getElementById("div-add-" + response.FieldID).classList.add("d-none");
      $('#div-' + response.FieldID).html(response.Value);

    } else {
      document.getElementById("div-edit-" + response.FieldID).classList.add("d-none");
      document.getElementById("div-add-" + response.FieldID).classList.remove("d-none");
    }
  }
}

function confirmDeleteModal(title, content, action) {

  $('#confirm-delete-modal-title').html(title);
  $('#confirm-delete-modal-form').attr("action", action);
  $('#confirm-delete-modal-content').html(content);

  //show the modal
  $('#confirm-delete-modal').modal('show');
}
