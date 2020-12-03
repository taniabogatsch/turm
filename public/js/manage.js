/* This file comprises js functions required at the manage courses page. */

function openDownloadModal(ID) {
  $('#download-course-modal-ID').val(ID);
  $('#download-course-modal').modal('show');
}

function uploadFeedback() {
  const filepath = $('#custom-file-upload').val();
  let path = filepath.split("/");
  if (filepath.includes("\\")) {
    path = filepath.split("\\");
  }
  $('#file-upload-success').removeClass("d-none");
  $('#file-upload-success').html(uploadMsg + " " + path[path.length - 1]);
}

function showChosenOption() {
  let option = $('#select-option').children("option:selected").val();
  //show additional content depending on the chosen option
  if (option == 1) {
    $('#file-upload-section').addClass("d-none");
    $('#search-draft-section').removeClass("d-none");
    $("#custom-file-upload").prop('required', false);
  } else if (option == 2) {
    $('#file-upload-section').removeClass("d-none");
    $('#search-draft-section').addClass("d-none");
    $("#custom-file-upload").prop('required', true);
  } else {
    $('#file-upload-section').addClass("d-none");
    $('#search-draft-section').addClass("d-none");
    $("#custom-file-upload").prop('required', false);
  }
}

function submitDownloadCourse() {

  $('#download-course-modal').modal('hide');
  $('#download-course-modal-form').submit();
}

function openDuplicateModal(ID) {

  $('#duplicate-course-ID').val(ID);
  $('#duplicate-course-modal').modal('show');
}
