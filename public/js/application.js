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

//groupsChangeIcon adjusts the group collapse icons according to the collapse state
function groupsChangeIcon(id) {

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
