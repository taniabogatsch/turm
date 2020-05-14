/* This file comprises manage courses javascript functions that must be assembled by the template engine. */

function getActiveCourses() {
  $.get('{{url "ManageCourses.GetActive"}}', {
  }, function(data) {
    $('#get-active-courses').html(data);
  })
}

function getDrafts() {
  $.get('{{url "ManageCourses.GetDrafts"}}', {
  }, function(data) {
    $('#get-drafts').html(data);
  })
}

function getExpiredCourses() {
  $.get('{{url "ManageCourses.GetExpired"}}', {
  }, function(data) {
    $('#get-expired-courses').html(data);
  })
}
