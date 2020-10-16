$( document ).ready(function() {
    console.log('index loaded')
    console.log('login callback url: ', loginCallbackURL())
});

function loginCallbackURL() {
    return location.protocol + '//' + location.host + '/tokenexchange'
}

function stravaLogin() {
    window.location = "https://www.strava.com/oauth/authorize" + 
        "?scope=" + encodeURIComponent('activity:read') +
        "&client_id=" + encodeURIComponent($( '#strava_client_id' )[0].value) +
        "&redirect_uri=" + encodeURIComponent(loginCallbackURL()) +
        "&response_type=" + encodeURIComponent('code') +
        "&approval_prompt=" + encodeURIComponent('auto')
}

$("#login").click(stravaLogin);
