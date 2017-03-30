window.onload = function() {
    var deck
    document.getElementById("room").focus()

    if (typeof(localStorage) !== "undefined") {
        if ( deck = localStorage.getItem("deck") ) {
            document.getElementById("deck").setAttribute("value", deck)
        }
    }
}
