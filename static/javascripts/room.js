var Sibyl = function() {
    this.inReveal = false

    if (!window["SibylConfig"] || !SibylConfig.Token) {
        this.addToConsole("Could not create room.")
        return
    } else if (!("WebSocket" in window)) {
        this.addToConsole("Your browser does not support Web Sockets.")
        return
    }

    this.deck = null
    this.room = SibylConfig.Room
    this.token = SibylConfig.Token
    this.topic = null
    this.lastConnectAttempt = 0

    this.connectToWebSocket()
    this.setupBindings()
}

Sibyl.prototype.setupBindings = function() {
    var self = this,
        $cards = $("#cards"),
        $topic = $("#topic"),
        $currentUser = $("#current-username"),
        $currentUser = $("#current-username")

        $("#room-url").html(document.location.href)
        $("#copy-url").click(function(e) {
        var x = e.clientX + document.body.scrollLeft,
            y = e.clientY + document.body.scrollTop,
            ok, msg = "link copied!",
            $copydiv,
            $input = $("<input>").css({
                position: "absolute",
                top: y,
                left: x,
                opacity: 0
            }).appendTo("body").val(document.location.href).focus().select()

        try {
            document.execCommand("copy")
        } catch(e) {
                msg = "browser does not support copy command"
        }

        $input.remove()

        $copydiv = $("<div>").html(msg).addClass("message").css({ top: y, left: x }).appendTo("body")
        setTimeout(function() {
            $copydiv.fadeTo("slow", 0, function() {
                $copydiv.remove()
            })
        }, 1000)

        return false;
    })

    $("#reveal").click(function() {
        if ($cards.find("span.card-facedown").length > 0) {
            self.send("reveal")
        }

        return false;
    })

    $("#reset").click(function() {
        self.send("reset")
        return false;
    })

    $(".decks a").click(function() {
        self.send("deck", { deck: $(this).attr("data-name") })
        return false
    })

    var textToInput = function(action, $text, value, inputClassName, maxLength) {
        var $form = $("<form>"),
            $input = $("<input>").attr("type", "text").attr("maxlength", maxLength).val(value).addClass(inputClassName),
            showText = function() {
                $text.show()
                $form.remove()
            }

        $input.blur(function() {
            showText()
        })

        $input.keydown(function(e) {
            if (e.keyCode == 27) { // escape key
                showText()
            }
        })

        $form.append($input)
        $text.hide()
        $text.parent().prepend($form)
        $form.submit(function() {
            // JavaScript (ES5) doesn't have support for unicode categories. We'll ensure all characters are
            // printable on the server
            if ($input.val().match(/\w/)) {
                $text.text( $input.val() )
                self.send(action, { value: $input.val() })
            }

            showText()
            return false
        })

        $input.focus().select()
        return false
    }

    $currentUser.click(function() {
        textToInput("username", $currentUser, self.username, "current-username-edit", SibylConfig.UsernameMaxLength)
    })

    $topic.click(function() {
        textToInput("topic", $topic, self.topic, "topic-edit", SibylConfig.TopicMaxLength)
    })

    $(window).on("beforeunload", function() {
        self.disconnect()
    })
}

Sibyl.prototype.updateBoard = function(data) {
    var n, i,
        $cards = $("#cards"),
        $myHand = $("#my-hand"),
        $topic = $("#topic"),
        $username = $("#current-username"),
        self = this,
        $myCard,
        $span,
        $div,
        deck

    this.username = data.username
    $username.text(this.username)

    this.topic = data.topic
    $topic.text(this.topic)

    this.inReveal = data.reveal

    if (data.reset) {
        $myHand.find("a").removeClass("chosen")
    }

    if ( !this.deck || this.deck != data.deck ) {
        this.deck = data.deck
        this.storeItem("deck", this.deck)
        deck = SibylConfig.Decks[this.deck]

        $myHand.html("")
        n = deck.cards.length
        for (i = 0; i < n; i++) {
            $myCard = $("<a>").attr("href", "#").attr("data-index", i)
            $myCard.append($("<span>").addClass("card").addClass("card-flipped").html(deck.cards[i]))
            $myHand.append($myCard)
        }

        $myHand.find("a").click(function() {
            var card = parseInt($(this).attr("data-index"), 10)
            if (!self.inReveal) {
                $myHand.find("a").removeClass("chosen")
                $(this).addClass("chosen")

                self.send("select", { card: card, deck: self.deck })
            }

            return false
        })
    }

    $cards.html("")

    var playerIDsToCards = {}
    n = data.cards.length
    for (i = 0; i < n; i++) {
        playerIDsToCards[data.cards[i].playerID] = data.cards[i].card
    }

    var playerIDs = []
    for (i in data.players) {
        if (data.players.hasOwnProperty(i)) {
            playerIDs.push(i)
        }
    }

    playerIDs.sort(function(a,b) {
        return data.players[a].localeCompare(data.players[b])
    })

    n = playerIDs.length
    for (i = 0; i < n; i++) {
        var playerID = playerIDs[i]

        $div = $("<div>").addClass("card")

        if ( playerID in playerIDsToCards ) {
            $span = $("<span>")
            $span.html(SibylConfig.Decks[this.deck].cards[ playerIDsToCards[playerID] ])
            $span.addClass("card")

            if (this.inReveal) {
                $span.addClass("card-flipped")
            } else {
                $span.addClass("card-facedown")
            }

            $div.append($span)

            $span = $("<span>").addClass("player-name").text(data.players[playerID])
            $div.append($span)
        } else {
            $div = $("<div>").addClass("card")
            $div.append($("<span>").addClass("card").addClass("card-blank").html("?"))
            $div.append($("<span>").addClass("player-name").text(data.players[playerID]))
        }

        $cards.append($div)
    }
}

Sibyl.prototype.connectToWebSocket = function(isRetry) {
    var self = this,
        url = (window.location.protocol == "https:" ? "wss://" : "ws://") + window.location.host + "/ws?room=" + encodeURIComponent(this.room) + "&token=" + encodeURIComponent(this.token),
        conn = new WebSocket(url),
        isOpen = false

    conn.onopen = function(evt) {
        isOpen = true
        isRetry = false
        self.addToConsole("Connected.")
        setTimeout(function() {
            if (isOpen) {
                self.showGame()
            }
        }, 250);
    }
    conn.onerror = function(evt) {
        self.addToConsole("Error. Lost connection.")
        self.showConsole()
    }
    conn.onclose = function(evt) {
        var now

        isOpen = false
        if (isRetry) {
            self.addToConsole("Server may be offline.")
        } else {
            self.addToConsole("Server disconnected.")
        }

        self.showConsole()

        if (!isRetry) {
            now = new Date().getTime() / 1000
            if ( now - self.lastConnectAttempt < 10 ) {
                self.addToConsole('Having an issue? Try using https: <a href="https://' + window.location.host + window.location.pathname + '">https://' + window.location.host + window.location.pathname + '</a>')
                return
            }
            self.lastConnectAttempt = now

            setTimeout(function() {
                self.addToConsole("Attempting to reconnect...")

                setTimeout(function() {
                    self.connectToWebSocket(true)
                }, 2500)
            }, 250)
        }
    }
    conn.onmessage = function(evt) {
        var data = JSON.parse(evt.data)
        if (data.error) {
            self.disconnect()
            self.addToConsole(data.error)
            self.showConsole()
        } else {
            self.updateBoard(data)
        }
    }

    this.conn = conn
}

Sibyl.prototype.disconnect = function() {
    this.addToConsole("Disconnected.")
    this.conn.onclose = function() { }
    this.conn.close(1000, "closing ok")
}

Sibyl.prototype.addToConsole = function(msg) {
    $("section.console div.block").append("<br>" + msg)
}

Sibyl.prototype.showGame = function() {
    $("section.console").hide()
    $("section.game").fadeIn("slow")
}

Sibyl.prototype.showConsole = function() {
    $("section.game").hide()
    $("section.console").show()
}

Sibyl.prototype.send = function(action, opts) {
    opts = opts ? opts : {}
    this.conn.send(JSON.stringify({
        action: action,
        card: opts.card || null,
        deck: opts.deck || null,
        room: this.room,
        token: this.token,
        value: opts.value || null
    }))
}

Sibyl.prototype.storeItem = function(key, value) {
    if (typeof(localStorage) !== "undefined") {
        try { localStorage.setItem(key, value) }
        catch (e) { }
    }
}

$(function() {
    var p = new Sibyl()
})
