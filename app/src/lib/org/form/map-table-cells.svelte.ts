import type { SchemaObjectValue, SchemaValue } from '@sjsf/form/core';
import { isQuietEmptyCell, writeObjectRowMapCell } from './object-row-weigh.js';

type ParentValue = SchemaObjectValue | null | undefined;

export type MapTableCellsOptions = {
	getValue: () => ParentValue;
	/** Whole-map write — used to drop a draft key (objCtx.set(undefined) leaves own-property undefined). */
	setValue: (next: ParentValue) => void;
	/** sjsf object-context set for non-empty row writes. */
	setRow: (rowKey: string, cell: SchemaObjectValue | undefined) => void;
};

/**
 * mapTable scalar bind: omit↔omit quiet; stock clear (undefined/null) drops the
 * draft key; enum values via setRow. Never writes JSON null as Domain clear.
 */
export class MapTableCells {
	#getValue: () => ParentValue;
	#setValue: (next: ParentValue) => void;
	#setRow: (rowKey: string, cell: SchemaObjectValue | undefined) => void;

	constructor(options: MapTableCellsOptions) {
		this.#getValue = options.getValue;
		this.#setValue = options.setValue;
		this.#setRow = options.setRow;
	}

	setScalarCell = (rowKey: string, cell: SchemaValue | undefined): void => {
		const current = this.#getValue()?.[rowKey];
		// omit stays omit — do not invent a key on quiet mount/bind
		if (current === undefined && cell === undefined) return;

		if (cell === undefined || cell === null) {
			if (current === undefined) return;
			const parent = this.#getValue();
			const out = {
				...(parent && typeof parent === 'object' ? parent : {})
			} as Record<string, unknown>;
			delete out[rowKey];
			this.#setValue(
				Object.keys(out).length === 0 ? undefined : (out as SchemaObjectValue)
			);
			return;
		}

		if (Object.is(current, cell)) return;
		this.#setRow(rowKey, cell as SchemaObjectValue);
	};

	setCell = (rowKey: string, column: string, cell: SchemaValue | undefined): void => {
		const value = this.#getValue();
		const current = (value?.[rowKey] as SchemaObjectValue | undefined)?.[column];
		if (isQuietEmptyCell(current) && isQuietEmptyCell(cell)) return;
		if (Object.is(current, cell)) return;
		const row = {
			...((value?.[rowKey] as SchemaObjectValue | undefined) ?? {})
		} as Record<string, unknown>;
		this.#setRow(
			rowKey,
			writeObjectRowMapCell(row, column, cell) as SchemaObjectValue | undefined
		);
	};
}
