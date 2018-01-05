window.onload = function() {
    var deck, quickCreateLink, quickCreateForm
    document.getElementById("room").focus()

	quickCreateForm = document.getElementById("create-room-quick")

	// handle the link on the index page when the user tries to join a non-existent page
	if (quickCreateForm) {
		quickCreateForm.getElementsByTagName("a")[0].onclick = function(e) {
			e.preventDefault && e.preventDefault()
			e.stopPropagation && e.stopPropagation()

			quickCreateForm.submit()

			return false
		}
	}

    if (typeof(localStorage) !== "undefined") {
        if ( deck = localStorage.getItem("deck") ) {
            document.getElementById("deck").setAttribute("value", deck)
        }
    }
}
