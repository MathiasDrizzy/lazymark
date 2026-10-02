-- lazymark editor plugin (managed by `lazymark editor-plugins`; remove it with `lazymark editor-plugins uninstall micro`)
--
-- Inserts the markdown reference of the image you have copied (a screenshot or an image
-- file copied in the file manager) at the cursor: ![](assets/…). The image is saved in
-- the note's assets/ folder by `lazymark paste`.
--
-- Key:     Alt-i (bound only if it is free: TryBindKey with overwrite=false)
-- Command: > pasteimage   (Ctrl-e, then pasteimage)
--
-- Plugin API: https://github.com/zyedidia/micro/blob/master/runtime/help/plugins.md
VERSION = "1.0.0"

local micro = import("micro")
local config = import("micro/config")
local shell = import("micro/shell")
local buffer = import("micro/buffer")

function pasteImage(bp)
	local path = bp.Buf.AbsPath
	if path == nil or path == "" then
		micro.InfoBar():Error("lazymark: save the note first, so the image can go in its assets/ folder")
		return
	end
	local out, err = shell.ExecCommand("lazymark", "paste", path)
	if err ~= nil then
		local msg = (tostring(out):gsub("%s+$", ""))
		if msg == "" then
			msg = tostring(err)
		end
		micro.InfoBar():Error(msg)
		return
	end
	local ref = (out:gsub("%s+$", ""))
	bp.Buf:Insert(buffer.Loc(bp.Cursor.X, bp.Cursor.Y), ref)
	micro.InfoBar():Message("lazymark: " .. ref)
end

-- true if the user already binds Alt-i in bindings.json (TryBindKey does not tell us)
local function altITaken()
	local f = io.open(config.ConfigDir .. "/bindings.json", "r")
	if f == nil then
		return false
	end
	local data = f:read("*a")
	f:close()
	return data ~= nil and string.find(data, '"Alt%-i"') ~= nil
end

function init()
	config.MakeCommand("pasteimage", pasteImage, config.NoComplete)
	config.TryBindKey("Alt-i", "lua:lazymark.pasteImage", false)
end

function postinit()
	if altITaken() then
		-- Alt-i already has a binding of yours: it is not replaced; the command keeps working
		micro.InfoBar():Message("lazymark: Alt-i is already bound; paste images with the command > pasteimage")
	end
end
