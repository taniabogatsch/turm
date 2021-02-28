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

//showToast loads an message into the toast and shows it
function showToast(content, type) {

  if (type == "success") {
    $("#toast-title").html($("#icon-flash-check").html());
  } else {
    $("#toast-title").html($("#icon-flash-alertCircle").html());
  }

  $("#toast-title").attr("class", "mr-auto text-" + type);
  $("#toast-content").html(content);
  $("#toast-content").attr("class", "text-" + type);
  $("#feedback-toast").toast('show');
}

//confirmPOSTModal confirms the execution of a POST action
//this is mostly used to confirm deletions
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

//openCategoryModal shows the modal to insert/update a category
function openCategoryModal(table, ID, name, action, title) {

  $('#admin-category-modal-form').attr("action", action);
  $('#admin-category-modal-title').html(title);
  $('#admin-category-modal-ID').val(ID);
  $('#admin-category-modal-table').val(table);
  $('#admin-category-modal-name').val(name);

  //show the modal
  $('#admin-category-modal').modal('show');
}

//openEntryModal shows the modal to insert/update a help page entry
function openEntryModal(ID, action, title, isFAQ, val1ID, val2ID,
  p1, p2, categoryID) {

  $('#admin-entry-modal-form').attr("action", action);
  $('#admin-entry-modal-title').html(title);
  $('#admin-entry-modal-ID').val(ID);
  $("#admin-entry-modal-category").val(categoryID);

  $('#admin-entry-modal-table').val(isFAQ);

  if (isFAQ) { //FAQ specifc setup

    //set the content names
    $('#admin-entry-modal-content1-name').html(p1);
    $('#admin-entry-modal-content2-name').html(p2);
    //show the second text area and content name
    $('#admin-entry-modal-content2').removeClass("d-none");
    $('#admin-entry-modal-content2-name').removeClass("d-none");
    //set the correct names of both text areas
    $('#admin-entry-modal-value1').attr("name", "entry.Question");
    $('#admin-entry-modal-value2').attr("name", "entry.Answer");
    //text area 2 requires input
    $('#admin-entry-modal-value2').attr("required", true);
    //set the content of both editors
    if (val1ID != "") {
      quill1.root.innerHTML = $('#' + val1ID).html();
    } else {
      const textArea = document.getElementById("admin-entry-modal-value1");
      textArea.setCustomValidity("Please provide a text.");
      quill1.root.innerHTML = "<p><br></p>";
    }
    if (val2ID != "") {
      quill2.root.innerHTML = $('#' + val2ID).html();
    } else {
      const textArea = document.getElementById("admin-entry-modal-value2");
      textArea.setCustomValidity("Please provide a text.");
      quill2.root.innerHTML = "<p><br></p>";
    }

  } else { //news feed specifc setup

    //set the content name
    $('#admin-entry-modal-content1-name').html(p1);
    //hide the second text area and content name
    $('#admin-entry-modal-content2').addClass("d-none");
    $('#admin-entry-modal-content2-name').addClass("d-none");
    //set the correct name of the text area
    $('#admin-entry-modal-value1').attr("name", "entry.Content");
    //set the editor content
    if (val1ID != "") {
      quill1.root.innerHTML = $('#' + val1ID).html();
    } else {
      const textArea = document.getElementById("admin-entry-modal-value1");
      textArea.setCustomValidity("Please provide a text.");
      quill1.root.innerHTML = "<p><br></p>";
    }
  }

  //show the modal
  $('#admin-entry-modal').modal('show');
}

//detectTextFieldChange detects changes in a quill text field area
function detectTextFieldChange(source, textField, text) {
  if (source == 'user') {
    const textArea = document.getElementById(textField);
    if (text != "<p><br></p>" && text != "") {
      $('#' + textField).val(text);
      textArea.setCustomValidity('');
    } else {
      textArea.setCustomValidity("Please provide a text.");
    }
  }
}

//openAdminGroupModal shows the modal to insert/update a group
function openAdminGroupModal(ID, parentID, inheritsLimits, action, title,
  value, childHasLimits, courseLimits) {

  $('#admin-group-modal-form').attr("action", action);
  $('#admin-group-modal-title').html(title);
  $('#admin-group-modal-ID').val(ID);
  $('#admin-group-modal-parentID').val(parentID);
  $('#admin-group-modal-name').val(value);

  $('#admin-group-modal-courseLimits').attr("disabled", (inheritsLimits || childHasLimits));
  if (inheritsLimits || childHasLimits) {
    $('#admin-group-modal-info-1').removeClass("d-none");
    $('#admin-group-modal-info-2').addClass("d-none");
  } else {
    if (courseLimits != 0) {
      $('#admin-group-modal-courseLimits').val(courseLimits);
    }
    $('#admin-group-modal-info-1').addClass("d-none");
    $('#admin-group-modal-info-2').removeClass("d-none");
  }

  $('#admin-group-modal').modal('show');
}

function enterEnrollDataModal(action, msg, ID, hasKey, hasComments) {

  $('#enter-enrollment-data-modal-form').attr("action", action);
  $('#enter-enrollment-data-modal-ID').val(ID);
  $('#enter-enrollment-data-modal-btn').html(msg);

  if (hasKey) {
    $('#enter-enrollment-data-modal-key').removeClass("d-none");
  } else {
    $('#enter-enrollment-data-modal-key').addClass("d-none");
  }

  if (hasComments) {
    $('#enter-enrollment-data-modal-comment').removeClass("d-none");
  } else {
    $('#enter-enrollment-data-modal-comment').addClass("d-none");
  }

  //show the modal
  $('#enter-enrollment-data-modal').modal('show');
}

function openCommentsModal(ID) {

  $('#comments-modal-content').html($('#event-comments-' + ID).html());
  $('#comments-modal').modal('show');
}

function bookSlotModal(courseID, calendarEventID, date, year, weekDay, monday, day) {

  $('#book-slot-modal-course-ID').val(courseID);
  $('#book-slot-modal-ID').val(calendarEventID);
  $('#book-slot-modal-date').val(date)
  $('#book-slot-modal-date-div').html(date);
  $('#book-slot-modal-year').val(year);
  $('#book-slot-modal-monday').val(monday);
  $('#book-slot-modal-day').val(day);

  $('#book-slot-modal-time-spans').html($('#pretty-time-spans-' + weekDay).html());

  //show the modal
  $('#book-slot-modal').modal('show');
}

function searchCourse(valDiv, resultDiv, dropdownID, action) {

  let dropdown = document.getElementById(resultDiv);
  let contains = dropdown.classList.contains("show");

  let value = $('#' + valDiv).val();
  if (value != "") {
    $.get(action, {
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

function searchCourseDrafts(valDiv, resultDiv, action) {

  let result = document.getElementById(resultDiv);

  let value = $('#' + valDiv).val();

  if (value != "") {

    $.get(action, {
      "value": value
    }, function(data) {
      $('#' + resultDiv).html(data);
    })

    result.classList.remove("d-none");

  } else {

    result.classList.add("d-none");
  }
}

function selectCourseIDOfDraft(ID) {

  $('#new-course-input-ID').val(ID);

  let items = document.getElementsByClassName("new-course-list-item");
  for (let i = 0; i < items.length; i++) {
    items[i].classList.remove("active");
  }
  document.getElementById('new-course-list-item-' + ID).classList.add("active");
}

function openDownloadModal(ID) {
  $('#download-course-modal-ID').val(ID);
  $('#download-course-modal').modal('show');
}

function submitDownloadCourse() {

  $('#download-course-modal').modal('hide');
  $('#download-course-modal-form').submit();
}

function openDuplicateModal(ID) {

  $('#duplicate-course-ID').val(ID);
  $('#duplicate-course-modal').modal('show');
}

function renderCalendarEvent(ID, courseID, shift, monday, action, isMobile) {

  let day = 0;
  if (isMobile) {
    day = currentDay;
  }

  $.get(action, {
    "ID": ID,
    "courseID": courseID,
    "shift": shift,
    "monday": monday,
    "day": day,
  }, function(data) {
    $('#calendar-event-' + ID).html(data);
  })
}

function unsubFromSlot(slotID, eventID, courseID, monday, action, day) {

  $.get(action, {
    "slotID": slotID,
    "eventID": eventID,
    "courseID": courseID,
    "monday": monday,
    "day": day,
  }, function(data) {
    $('#calendar-event-' + eventID).html(data);
  })
}

function disableEnrollmentButtons() {

  $(".enroll-btn").each(function() {
    $(this).attr("href", "#no-scroll");
    $(this).addClass('disabled');
  });
}

function hideEnrollInfoMessages() {

  $(".enroll-info").each(function() {
    $(this).addClass("d-none");
  });
}

function getPreviousDay(ID, courseID, monday, action) {

  //leaving the current week
  if (currentDay == 0) {
    renderCalendarEvent(ID, courseID, -1, monday, action, true);

  } else { //previous day of current week

    currentDay -= 1;
    getDay();
  }
}

function getNextDay(ID, courseID, monday, action) {

  //leaving the current week
  if (currentDay == 6) {
    renderCalendarEvent(ID, courseID, 1, monday, action, true);

  } else { //previous day of current week

    currentDay += 1;
    getDay();
  }
}

function getDay() {

  //hide all elements
  $(".day-elem").each(function() {
    $(this).addClass("d-none");
  });

  //only show elements of the selected day
  $('#day-header-' + currentDay).removeClass('d-none');
  $('#day-tmpls-' + currentDay).removeClass('d-none');
  $('#day-schedule-' + currentDay).removeClass('d-none');
}
