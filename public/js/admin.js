/* This file comprises js functions required at the administration page. */

//called if the search value changes or if the 'inactive users'-checkbox is changed
function reactToInput() {

  document.getElementById("search-form").classList.add('was-validated');
  const value = $('#user-search-input').val();

  if (value.length > 2) { //search matching users
    const searchInactive = $('#search-inactive').is(':checked');
    search(value, searchInactive);

  } else if (value.length == 0) {
    $('#user-search-results').html("");
    document.getElementById("search-form").classList.remove('was-validated');

  } else {
    $('#user-search-results').html("");
  }
}

//render user details of a user
function userDetails(ID, action) {

  $('#v-pills-users-tab').tab('show');

  $.get(action, {
    "ID": ID
  }, function(data) {
    $('#nav-pill-content-users').html(data);
  });

  $('#admin-load-user-modal').modal('hide');
}

//openLoadUserModal opens the load user modal and refreshes its content
function openLoadUserModal() {

  const value = $('#user-search-input').val();
  if (value.length > 2) {
    reactToInput();
  }

  $('#admin-load-user-modal').modal('show');
}
