<script>
  export let name = "AllergyIntolerance";
  export let data = "";
  const resources = data.entry.map(entry => entry.resource);
  const entry = resources.find(resource => resource.resourceType === name);
</script>

<h4>{name} details</h4>
<pre>{JSON.stringify(entry, null, 2)}</pre>