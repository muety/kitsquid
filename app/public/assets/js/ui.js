$(() => {
    const inputSignupPrefix = $('#input-signup-prefix'),
        inputSignupSuffix = $('#input-signup-suffix'),
        inputSignupUser = $('#input-signup-user'),
        inputSignupPassword = $('#input-signup-password'),
        inputSignupGender = $('input[type=radio][name="gender"]'),
        formSignup = $('#form-signup'),
        imgAvatar = $('#img-avatar')

    $(window).click(function () {
        toggleLogoutButton(false)
        toggleLoginButton(false)
    })

    // Main Alert
    if (localStorage.getItem('hide_main_alert')) {
        closeMainAlert()
    }

    // Signup
    if (window.hasOwnProperty('signupConfig')) {
        inputSignupSuffix.change(() => {
            inputSignupPrefix.attr('pattern', window.signupConfig[inputSignupSuffix.val()][1])
            inputSignupPrefix.attr('placeholder', window.signupConfig[inputSignupSuffix.val()][0])
            inputSignupPassword.attr('pattern', window.signupConfig[inputSignupSuffix.val()][2])
        })
    }

    if (formSignup) {
        formSignup.submit(() => {
            let user = `${inputSignupPrefix.val()}@${inputSignupSuffix.val()}`
            inputSignupUser.val(user)
        })

        let debounce = null
        inputSignupPrefix.keyup(() => {
            if (debounce) clearTimeout(debounce)
            debounce = setTimeout(updateAvatar, 250)
        })
        inputSignupGender.change(updateAvatar)
    }

    function updateAvatar() {
        let re = new RegExp(inputSignupPrefix.attr('pattern'), 'i')
        let valid = inputSignupPrefix.val().match(re)
        let gender = $('input[type=radio][name="gender"]:checked').val()
        let avatarUrl = valid
            ? `https://avatars.dicebear.com/v2/${gender}/${inputSignupPrefix.val()}.svg`
            : '/assets/images/unknown.png'
        imgAvatar.attr('src', avatarUrl)
    }
})

function toggleSidebar() {
    let open = !$('#sidebar').hasClass('hidden')
    $('#sidebar').toggleClass('hidden', open)
    $('#main-content').toggleClass('hidden', !open)
}

function toggleLogoutButton(forceState) {
    let open = !$('#btn-logout').hasClass('hidden')
    $('#btn-logout').toggleClass('hidden', forceState === undefined ? open : !forceState)
}

function toggleLoginButton(forceState) {
    let open = !$('#btn-login').hasClass('hidden')
    $('#btn-login').toggleClass('hidden', forceState === undefined ? open : !forceState)
}

function closeAlert(prefix, i) {
    $(`#${prefix}-${i}`).addClass('hidden')
}

function logout() {
    $('#form-logout').submit()
}

function closeMe(event) {
    $(event.target).addClass('hidden')
}

function closeParent(event) {
    closeMe({target: event.target.parentNode})
}

function closeMainAlert() {
    $('#alert-main').addClass('hidden')
    localStorage.setItem('hide_main_alert', 'true')
}

function toggleBookmarkEvent(id) {
    let indicator = $(`#indicator-bookmark-${id}`)
    let indicatorEmpty = $(`#indicator-bookmark-${id}-empty`)

    fetch(`/api/event/${id}/bookmark`, {method: 'PUT'})
        .then(response => {
            if (!response.ok) {
                return response.json().then((data) => {
                    showSnackbar(`Fehler: ${data.error}`)
                })
            } else if (response.status === 201) {
                indicator.removeClass('hidden').addClass('block')
                indicatorEmpty.addClass('hidden').removeClass('block')
                showSnackbar('Zu Favoriten hinzugefügt ...')
            } else if (response.status === 204) {
                indicator.addClass('hidden').removeClass('block')
                indicatorEmpty.removeClass('hidden').addClass('block')
                showSnackbar('Aus Favoriten entfernt ...')
            }
        })
        .catch(() => {
            showSnackbar('Fehler: Veranstaltung konnte nicht zu den Favoriten hinzugefügt werden')
        })
}

function showSnackbar(text) {
    let sb = $('#snackbar')
    sb.text(text)
    sb.removeClass('hidden')
    setTimeout(() => {
        sb.addClass('hidden')
    }, 2000)
}