/* This file comprises admin javascript functions that must be assembled by the template engine. */

function search(value, searchInactive) {
  $.get("{{url "Admin.SearchUser"}}", {
    "value": value,
    "searchInactive": searchInactive
  }, function(data) {
    $('#user-search-results').html(data);
  });
}
