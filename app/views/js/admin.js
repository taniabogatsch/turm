/* This file comprises admin javascript functions that must be assembled by the template engine. */

function userDetails(ID) {
  $.get("{{url "Admin.UserDetails"}}", {
    "ID": ID
  }, function(data) {
    $('#user-details-modal-content').html(data);
  });
  $('#user-details-modal').modal('show');
}

function search(value, searchInactive) {
  $.get("{{url "Admin.SearchUser"}}", {
    "value": value,
    "searchInactive": searchInactive
  }, function(data) {
    $('#user-search-results').html(data);
  });
}
