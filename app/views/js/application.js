/* This file comprises javascript functions (shared by different views of the application). */

//JavaScript for disabling form submissions if there are invalid fields
(function() {
  'use strict';
  window.addEventListener('load', function() {
    //fetch all the forms to which we want to apply custom bootstrap validation styles
    var forms = document.getElementsByClassName('needs-validation');
    //loop over them and prevent submission
    var validation = Array.prototype.filter.call(forms, function(form) {
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

//changeIcon adjusts the group collapse icons according to the collapse state
function changeIcon(id) {

  //get icons
  const iconRightClass = $("#icon-right-" + id).attr("class");
  const iconDownClass = $("#icon-down-" + id).attr("class");

  if (iconRightClass == "display-block") {
    $("#icon-right-" + id).attr("class", "display-none");
    $("#icon-down-" + id).attr("class", "display-block");
  } else {
    $("#icon-right-" + id).attr("class", "display-block");
    $("#icon-down-" + id).attr("class", "display-none");
  }
}

//load an error message into the toast and show it
function showErrorToast(content) {

  $("#toast-title").html($("#icon-flash-alertCircle").html());
  $("#toast-title").attr("class", "mr-auto color-danger");
  $("#toast-content").html(content);
  $("#toast-content").attr("class", "color-danger");
  $("#feedback-toast").toast('show');
}

//load a success message into the toast and show it
function showSuccessToast(content) {

  $("#toast-title").html($("#icon-flash-check").html());
  $("#toast-title").attr("class", "mr-auto color-success");
  $("#toast-content").html(content);
  $("#toast-content").attr("class", "color-success");
  $("#feedback-toast").toast('show');
}

//loads the groups template
function getGroups(prefix, div) {
  $.get('{{url "App.Groups"}}', {
    "prefix": prefix
  }, function(data) {
    $(div).html(data);
  })
}

function openGroupsNav() {
  getGroups("nav", '#nav-groups-modal-content');
  $('#nav-groups-modal').modal('show');
}

//hide all elements that are not to be seen by users without the respective authority
$(function() {

  if ({{.session.role}} == "admin") {
    $(".admin").each(function() {
      $(this).removeClass("display-none");
    });
  }

  if ({{.session.role}} == "creator") {
    $(".creator").each(function() {
      $(this).removeClass("display-none");
    });
  }

  if ({{.session.isEditor}} == "true") {
    $(".editor").each(function() {
      $(this).removeClass("display-none");
    });
  }

  if ({{.session.isInstructor}} == "true") {
    $(".instructor").each(function() {
      $(this).removeClass("display-none");
    });
  }
});

{{if $.session.userID}}
  {{if eq $.session.role "admin"}}

    function openCreateGroup(parentID, inheritsLimits) {

      $('#group-form').attr("action", '{{url "Admin.AddGroup"}}');
      $('#group-modal-title').html('{{msg $ "group.new"}}');
      $('#input-parentID').val(parentID);
      $('#input-name').val("");
      $('#group-confirm-btn').html('{{msg $ "button.add"}}');

      if (inheritsLimits) {
        $('#input-courseLimits').attr("disabled", true);
        $('#input-courseLimits').val("");
        $('#courseLimits-info').html('{{msg $ "group.inherits.limit.info"}}');
      } else {
        $('#input-courseLimits').attr("disabled", false);
        $('#input-courseLimits').val("");
        $('#courseLimits-info').html('{{msg $ "group.course.limit.x.info"}}');
      }

      $('#group-modal').modal('show');
    }

    function openEditGroup(ID, parentID, childHasLimits, inheritsLimits, name, courseLimits) {

      $('#group-form').attr("action", '{{url "Admin.EditGroup"}}');
      $('#group-modal-title').html('{{msg $ "group.edit"}}');
      $('#input-ID').val(ID);
      $('#input-parentID').val(parentID);
      $('#input-name').val(name);
      $('#group-confirm-btn').html('{{msg $ "button.edit"}}');

      if (courseLimits != 0) {
        $('#input-courseLimits').attr("disabled", false);
        $('#input-courseLimits').val(courseLimits);
        $('#courseLimits-info').html('{{msg $ "group.course.limit.x.info"}}');
      } else if (inheritsLimits || childHasLimits) {
        $('#input-courseLimits').attr("disabled", true);
        $('#input-courseLimits').val("");
        $('#courseLimits-info').html('{{msg $ "group.inherits.limit.info"}}');
      }

      $('#group-modal').modal('show');
    }

    function openDeleteGroup(ID, content) {
      $('#group-delete-content').html(content);
      $('#delete-input-ID').val(ID);
      $('#group-delete-modal').modal('show');
    }
  {{end}}
{{end}}
