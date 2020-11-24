/* This file comprises participants javascript functions that must be assembled by the template engine. */

function searchEntries(eventIdx, courseID, eventID, value) {

  $.get("{{url "Participants.SearchUser"}}", {
    "ID": courseID,
    "eventID": eventID,
    "value": value
  }, function(data) {
    $('#user-search-results-' + eventIdx).html(data);
  });
}
