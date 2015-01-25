$(document).ready(function() {
	$(".newQuestionForm").on("submit", function(event) {
		var data = {};
		$.each($(event.target).serializeArray(), function(i, field) {
			data[field.name] = field.value;
		});
		$.ajax({
			type: 'POST',
			url: '/newQuestion/',
			data: JSON.stringify(data),
		})
		.done(function(r, textStatus) {
			window.location.replace("/questions/" + r);
		})
		.fail(function(r) {
			console.log("fail: " + JSON.stringify(r));
			$("#newQuestionError").text("An error has occured.");
		});
		return false;
	});
});
