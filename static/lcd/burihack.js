'use strict';

(function(){

function get(line) {
	var pixState = document[line].src.charAt(
			(document[line].src.length) - 6);
	return pixState != "0";
}

function getC(line) {
	var pixState = document[line].src.charAt(
			(document[line].src.length) - 7);
	return pixState != "0";
}

function set(line) {
	if(!get(line)) { togImage(line) }
}

function clear(line) {
	if(get(line)) { togImage(line) }
}

function setC(line) {
	if(!getC(line)) { togImageC(line) }
}

function clearC(line) {
	if(getC(line)) { togImageC(line) }
}

function read(addr) {
	return 0x00;
}

function write(addr, val) {
	var r = addr - 0xdff0, i = 0;
	if(r == 0) {
		clear('rs');
	} else if(r == 1) {
		set('rs');
	} else {
		console.log('bad address:', addr);
		return;
	}

	for(i=0; i<8; i++) {
		if((val&(1<<i)) != 0) {
			set('d' + i);
		} else {
			clear('d' + i);
		}
	}

	clear('rw');
	setC('e');
	clearC('e');
	setC('rw');
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
