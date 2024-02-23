// OVLcurrentvisitors.js
//
// Creative Commons: Attribution/Share Alike/Non Commercial (cc) 2024 Maker Nexus
// By Jim Schrempp
//

document.addEventListener('DOMContentLoaded', function() {
    // document is loaded

    // add a listener to each link on the form so we can intercept the click event
    // and prevent the browser from following the link. Also stop a double click event.
    var links = document.querySelectorAll('a');
    if(links.length > 0) {
        links.forEach(function(link) {
            
            var handleClick = function(event) {
                // here's where each link comes when clicked

                // Prevent race condition where the user clicks the same
                // link again before the database is updated and the page has reloaded.
                event.currentTarget.removeEventListener('click', handleClick);

                event.preventDefault();

                // Create overlay to prevent second click
               var overlay = document.createElement('div');
               overlay.style.position = 'fixed';
               overlay.style.top = 0;
               overlay.style.left = 0;
               overlay.style.width = '100%';
               overlay.style.height = '100%';
               overlay.style.zIndex = 10000; // Make sure it's on top of everything
               document.body.appendChild(overlay);

                // call the URL
                var url = event.currentTarget.href;
                fetch(url)
                    .then(data => {console.log(data);
                        // reload the page to get new visitor list
                        window.location.reload();
                    })
                    .catch((error) => {
                        console.error('Error:', error);
                        allowClicks();
                    });

               allowClicks(function() {
                   // Remove overlay before reloading
                   document.body.removeChild(overlay);
               });     
            };

            // add the listener to the link
            link.addEventListener('click', handleClick);

        });
    } else {
        console.log('No links found');
    }
});
