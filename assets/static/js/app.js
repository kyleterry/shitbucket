$(document).ready(function() {
    $(".url-container").hover(function() {
        $(this).find(".url-actions").removeClass('hidden');
    },
    function() {
        $(this).find(".url-actions").addClass('hidden');
    });
});
