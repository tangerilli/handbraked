var Directory = Backbone.Model.extend({
    idAttribute: "Name"
});

var DirectoryView = Backbone.View.extend({
    events: {
        "click li.directory a": "clickDirectory"
    },
    initialize: function(options) {
        this.parent = options.parent;
        this.childViews = {};
    },
    render: function() {
        this.$el.html('');
        if(this.parent) {
            this.$el.append("<li class='directory'><a href='#' data-target='..'><i class='icon-folder-close'></i>..</a></li>");
        }
        _.each(this.model.get('Children'), function(dir) {
            this.$el.append("<li class='directory'><a href='#' data-target='" + dir.Name + "'><i class='icon-folder-close'></i>" + dir.Name + "</a></li>");
        }, this);
        _.each(this.model.get('Files'), function(file) {
            this.$el.append("<li class='file' data-path='" + file.Path + "'><i class='icon-file'></i><label class='checkbox'><input type='checkbox'>" + file.Name + "</label></li>");
        }, this);
        return this;
    },
    clickDirectory: function(evt) {
        var targetName = $(evt.currentTarget).attr('data-target');
        if(targetName == ".." && this.parent) {
            this.parent.render();
            return false;
        }
        var subdir = _.find(this.model.get('Children'), function(dir) {
            return dir.Name == targetName
        }, this);
        if(!subdir) return;

        if(!this.childViews[targetName]) {
            this.childViews[targetName] = new DirectoryView({
                model: new Directory(subdir),
                el: this.el,
                parent: this
            });
        }
        this.childViews[targetName].render();
        return false;
    }
})

var HandbrakeRouter = Backbone.Router.extend({
    routes: {
        "":"default",
    },
    default: function() {
        $.getJSON("/api/files/source", function(data) {
            var root = new Directory(data);
            var view = new DirectoryView({
                model: root,
                el: $("ul.files")
            });
            view.render();
        });
        $("button#queue").on("click", this.queueFiles);
    },
    queueFiles: function() {
        console.debug("Queue files");
        $("li.file").each(function(idx, el) {
            var checked = $("input", el).prop('checked');
            if(!checked) return;
            var path = $(el).attr('data-path');
            $.ajax({
                type: "POST", 
                url: "/api/files/queue", 
                data: '{"Path":"' + path + '"}', 
                contentType: "application/json",
                success: function() {
                    console.debug("Queued " + path);
                }
            });
        });
    }
});