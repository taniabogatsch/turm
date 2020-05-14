/* This file comprises edit course javascript functions that must be assembled by the template engine. */

function searchForList(value, searchInactive, listType, courseID) {
  $.get("{{url "EditCourse.SearchUser"}}", {
    "value": value,
    "searchInactive": searchInactive,
    "listType": listType,
    "ID": courseID
  }, function(data) {
    $('#change-user-list-modal-results').html(data);
  });
}
