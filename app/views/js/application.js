/* This file comprises javascript functions that must be assembled by the template engine. */

//getGroups loads the group template
function getGroups(prefix, div) {
  $.get('{{url "App.Groups"}}', {
    "prefix": prefix
  }, function(data) {
    $(div).html(data);
  })
}

//on-load function to hide all elements that are not to be seen by users without the respective authority
$(function() {

  if ({{.session.role}} == "admin") {
    $(".admin").each(function() {
      $(this).removeClass("d-none");
    });
  }

  if ({{.session.role}} == "creator") {
    $(".creator").each(function() {
      $(this).removeClass("d-none");
    });
  }

  if ({{.session.isEditor}} == "true") {
    $(".editor").each(function() {
      $(this).removeClass("d-none");
    });
  }

  if ({{.session.isInstructor}} == "true") {
    $(".instructor").each(function() {
      $(this).removeClass("d-none");
    });
  }

  //detect validation error
  if ({{if .errors}}true{{else}}false{{end}}) {
    showErrorToast($('#flash-errors').html());
  }
});

{{if $.session.userID}}
  {{if eq $.session.role "admin"}}

    function openCreateGroup(parentID, inheritsLimits) {

      $('#group-form').attr("action", '{{url "Admin.AddGroup"}}');
      $('#group-modal-title').html('{{msg $ "group.new"}}');
      $('#input-parentID').val(parentID);
      $('#input-name').val("");
      $('#group-confirm-btn').html('{{msg $ "button.create"}}');

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
