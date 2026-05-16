export namespace main {
	
	export class CSIFrame {
	    index: number;
	    raw: number[];
	    amplitude: number[];
	    phase: number[];
	
	    static createFrom(source: any = {}) {
	        return new CSIFrame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.raw = source["raw"];
	        this.amplitude = source["amplitude"];
	        this.phase = source["phase"];
	    }
	}

}

