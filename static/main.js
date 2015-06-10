'use strict';

(function(){

function read(addr) {
	return 0x8d;
}

function write(addr, val) {
	console.log('write', addr, val);
}

var ws = new WebSocket('ws://' + location.host + '/bus');
ws.onmessage = function(event) {
	var m = JSON.parse(event.data);

	if(m._type == 'write') {
		write(m.address, m.value);
	} else if(m._type == 'read') {
		ws.send(JSON.stringify({
			_type: 'readresp',
			address: m.address,
			value: read(m.address),
		}));
	} else {
		console.log('unknown msg:', m);
	}
}

})();
