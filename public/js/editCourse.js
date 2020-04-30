/* This file comprises js functions required to load the different modals for editing a course. */

//reactToListInput is called if the user search value changes
function reactToListInput() {

  //get the list type
  const listType = $('#change-user-list-modal-list').val();

  //validate the form
  document.getElementById("change-user-list-modal-form").classList.add('was-validated');

  //get the search value
  const value = $('#change-user-list-modal-search').val();

  if (value.length > 2) { //search matching users
    const searchInactive = $('#change-user-list-modal-checkbox').is(':checked');
    searchForList(value, searchInactive, listType);

  } else if (value.length == 0) { //no search value entered
    $('#change-user-list-modal-results').html("");
    document.getElementById("change-user-list-modal-form").classList.remove('was-validated');

  } else { //not enough characters
    $('#change-user-list-modal-results').html("");
  }
}

//submitList submits the action of the specified form
function submitList(ID) {
  //set the user ID
  $('#change-user-list-modal-input').attr("value", ID);
  $('#change-user-list-modal-form').submit();
}

//changeTextModal sets all fields of the add/edit text modal
function changeTextModal(title, textType, value, maxlength, validation) {

  //set the modal title
  $('#change-text-modal-title').html(title);
  //set the text type
  $('#change-text-modal-type').val(textType);
  //set name, value, placeholder, maxlength
  $('#change-text-modal-input').val(value);
  $('#change-text-modal-input').attr("placeholder", title);
  $('#change-text-modal-input').attr("maxlength", maxlength);
  $('#change-text-modal-validation').html(validation);
  //show the modal
  $('#change-text-modal').modal('show');
}

//openDeleteField sets all fields of the delete field modal
function openDeleteField(title, content, action) {

  //set the modal title
  $('#delete-field-modal-title').html(title);
  //set the action
  $('#delete-field-modal-form').attr("action", action);
  //set the content
  $('#delete-field-modal-content').html(content);
  //show the modal
  $('#delete-field-modal').modal('show');
}

//changeUserListModal sets all fields of the user list modal
function changeUserListModal(title, listType) {

  //reset the search
  $('#change-user-list-modal-search').val("");
  reactToListInput();

  //set the modal title
  $('#change-user-list-modal-title').html(title);
  //set the list type
  $('#change-user-list-modal-list').val(listType);
  //show the modal
  $('#change-user-list-modal').modal('show');
}

//openBoolModal sets all fields of the change bool modal
function openBoolModal(title, action, option1, option2, value, userID, listType) {

  //set the modal title
  $('#change-bool-modal-title').html(title);
  //set the action
  $('#change-bool-modal-form').attr("action", action);

  //set options and show the correct one
  $('#change-bool-modal-option-1').html(option1);
  $('#change-bool-modal-option-2').html(option2);
  $("#change-bool-modal-checkbox").prop("checked", value);
  if (value) {
    $('#change-bool-modal-option-1').removeClass("display-none");
    $('#change-bool-modal-option-2').addClass("display-none");
  } else {
    $('#change-bool-modal-option-1').addClass("display-none");
    $('#change-bool-modal-option-2').removeClass("display-none");
  }

  //set userID and listType
  $('#change-bool-modal-input').val(userID);
  $('#change-bool-modal-list').val(listType);

  //show the modal
  $('#change-bool-modal').modal('show');
}

//openTimestampModal sets all fields of the change timestamp modal
function openTimestampModal(title, info, type, timestamp) {

  //set the modal title
  $('#change-timestamp-modal-title').html(title);
  //set the modal info
  $('#change-timestamp-modal-info').html(info);
  //set the timestamp type
  $('#change-timestamp-modal-type').val(type);
  //set the date and time
  if (timestamp != "") {
    const timestampParts = timestamp.split(" ");
    $('#change-timestamp-modal-date').val(timestampParts[0]);
    $('#change-timestamp-modal-time').val(timestampParts[1]);
  }
  //show the modal
  $('#change-timestamp-modal').modal('show');
}

function openTextAreaModal(title, textType, valid) {

  //set the modal title
  $('#change-text-area-modal-title').html(title);
  //set the text type
  $('#change-text-area-modal-type').val(textType);
  //set content
  if (valid) {
    quill.root.innerHTML = $('#course-' + textType).html();
  } else {
    quill.root.innerHTML = "";
  }
  //show the modal
  $('#change-text-area-modal').modal('show');
}

//setTextAreaData sets the textarea of the changeTextModal
//according to the content in the quill editor
function setTextAreaData() {

  document.getElementById("change-text-area-modal-form").classList.add('was-validated');
  var textArea = document.getElementById("change-text-area-modal-content");

  var text = quill.root.innerHTML;

  if (text != "<p><br></p>") {
    $('#change-text-area-modal-content').val(text);
    textArea.setCustomValidity('');
    $('#change-text-area-modal-form').submit();
  } else {
    textArea.setCustomValidity("Please provide a text.");
  }
}

//submitChangeGroupModal sets the parent ID and submits the modal
function submitChangeGroupModal(parentID) {

  $('#change-group-modal-parentID').val(parentID);
  $('#change-group-modal-form').submit();
}
