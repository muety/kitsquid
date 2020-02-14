$(() => {
    const inputSignupPrefix = $('#input-signup-prefix'),
        inputSignupSuffix = $('#input-signup-suffix'),
        inputSignupUser = $('#input-signup-user'),
        inputSignupPassword = $('#input-signup-password'),
        formSignup = $('#form-signup')

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