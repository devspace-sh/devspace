window.addEventListener('load', function() {
    var el = document.getElementsByClassName('hljs')
    
    for (let i = 0; i < el.length; i++) {
        var lineNumber = 0
        
        el[i].innerHTML = el[i].innerHTML.replace(/(^|\n)/g, function() {
            lineNumber++;
            var lineId = 'block-' + i + '-line-' + lineNumber
            return arguments[1] + '<a class="line" href="#' + lineId + '" id="' + lineId + '">' + lineNumber + '</a>';
        })

        el[i].removeChild(el[i].lastChild)
    }
});
